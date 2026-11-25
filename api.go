package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/security"

	"github.com/floholz/wishlist/migrations"
)

// Capability API. A list is reachable through two random links:
//
//	/e/{edit_token} — full edit rights, never shows reservations
//	/s/{share_id}   — read-only + reserve, used by friends & family
//
// An account is optional: signed-in users own their lists (owner relation), so the
// edit link can always be recovered from the dashboard.

const (
	maxItemsPerList = 300
	maxLinks        = 10
	maxDashTokens   = 200
)

type itemJSON struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Links       []string `json:"links"`
	Photos      []string `json:"photos"`
	LinkPhoto   string   `json:"linkPhoto"` // fallback picture fetched from the links
	Position    int      `json:"position"`
	Reserved    *bool    `json:"reserved,omitzero"` // share view only
	// share view, signed in: reserved by this account (works without the browser key)
	ReservedByMe bool `json:"reservedByMe,omitzero"`
}

type listJSON struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Preset      string     `json:"preset"`
	Share       string     `json:"share"`
	Edit        string     `json:"edit,omitempty"` // edit view / dashboard only
	Owned       bool       `json:"owned"`          // has an account owner
	Mine        bool       `json:"mine"`           // owned by the requesting user
	Updated     string     `json:"updated"`
	Count       int        `json:"count"`
	Preview     []string   `json:"preview,omitempty"` // dashboard: first few item names
	Items       []itemJSON `json:"items,omitempty"`
}

func registerAPI(se *core.ServeEvent) {
	g := se.Router.Group("/api/wl")
	g.POST("/lists", createList)
	g.POST("/dashboard", dashboard)
	g.POST("/claim", claim).Bind(apis.RequireAuth("users"))

	g.GET("/e/{token}", withList(getEditList))
	g.PATCH("/e/{token}", withList(patchList))
	g.DELETE("/e/{token}", withList(deleteList))
	g.POST("/e/{token}/items", withList(createItem))
	g.PATCH("/e/{token}/items/{id}", withList(updateItem))
	g.DELETE("/e/{token}/items/{id}", withList(deleteItem))
	g.PUT("/e/{token}/order", withList(reorder))
	g.POST("/e/{token}/items/{id}/preview", withList(linkPreview))

	g.GET("/s/{share}", getShareList)
	g.POST("/s/{share}/items/{id}/reserve", reserve)
	g.DELETE("/s/{share}/items/{id}/reserve", unreserve)
}

// ---------- helpers ----------------------------------------------------------

func authUser(e *core.RequestEvent) *core.Record {
	if e.Auth != nil && e.Auth.Collection().Name == "users" {
		return e.Auth
	}
	return nil
}

func withList(h func(*core.RequestEvent, *core.Record) error) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		tok := e.Request.PathValue("token")
		if len(tok) < 20 {
			return uerr(404, "list_not_found", "List not found.").api()
		}
		list, err := e.App.FindFirstRecordByData("lists", "edit_token", tok)
		if err != nil {
			return uerr(404, "list_not_found", "List not found.").api()
		}
		return h(e, list)
	}
}

func listItems(app core.App, listID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("items", "list = {:l}", "position,created", 0, 0, dbx.Params{"l": listID})
}

func itemOf(e *core.RequestEvent, list *core.Record) (*core.Record, error) {
	it, err := e.App.FindRecordById("items", e.Request.PathValue("id"))
	if err != nil || it.GetString("list") != list.Id {
		return nil, uerr(404, "item_not_found", "Item not found.").api()
	}
	return it, nil
}

func toItem(r *core.Record) itemJSON {
	links := []string{}
	_ = r.UnmarshalJSONField("links", &links)
	return itemJSON{
		ID: r.Id, Name: r.GetString("name"), Description: r.GetString("description"),
		Links: links, Photos: r.GetStringSlice("photos"), Position: r.GetInt("position"),
		LinkPhoto: r.GetString("link_photo"),
	}
}

func toList(e *core.RequestEvent, l *core.Record, withEdit bool) listJSON {
	out := listJSON{
		Title: l.GetString("title"), Description: l.GetString("description"), Preset: l.GetString("preset"),
		Share: l.GetString("share_id"), Owned: l.GetString("owner") != "",
		Updated: l.GetDateTime("updated").String(),
	}
	if u := authUser(e); u != nil && l.GetString("owner") == u.Id {
		out.Mine = true
	}
	if withEdit {
		out.Edit = l.GetString("edit_token")
	}
	return out
}

// touch bumps the list's `updated` so the dashboard can sort by recent activity.
func touch(app core.App, list *core.Record) { _ = app.Save(list) }

func saveErr(e *core.RequestEvent, err error) error {
	var apiErr *router.ApiError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return uerr(400, "save_failed", "Could not save — please check your input.").api()
}

func cleanLinks(raw []string) ([]string, error) {
	out := []string{}
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !strings.Contains(s, "://") {
			s = "https://" + s
		}
		u, err := url.Parse(s)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(s) > 2000 {
			return nil, uerr(400, "invalid_link", "Invalid link: "+s, "url", s)
		}
		out = append(out, u.String())
	}
	if len(out) > maxLinks {
		return nil, uerr(400, "too_many_links", "Too many links.", "max", maxLinks)
	}
	return out, nil
}

func validPreset(p string) bool { return slices.Contains(migrations.Presets, p) }

// ---------- lists ------------------------------------------------------------

func createList(e *core.RequestEvent) error {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Preset      string `json:"preset"`
	}
	if err := e.BindBody(&body); err != nil {
		return uerr(400, "invalid_body", "Invalid request.").api()
	}
	if !validPreset(body.Preset) {
		body.Preset = "blank"
	}
	if body.Title = strings.TrimSpace(body.Title); body.Title == "" {
		body.Title = "My wishlist"
	}
	col, err := e.App.FindCollectionByNameOrId("lists")
	if err != nil {
		return err
	}
	l := core.NewRecord(col)
	l.Set("title", body.Title)
	l.Set("description", strings.TrimSpace(body.Description))
	l.Set("preset", body.Preset)
	l.Set("share_id", security.RandomString(10))
	l.Set("edit_token", security.RandomString(28))
	if u := authUser(e); u != nil {
		l.Set("owner", u.Id)
	}
	if err := e.App.Save(l); err != nil {
		return saveErr(e, err)
	}
	return e.JSON(http.StatusCreated, toList(e, l, true))
}

func getEditList(e *core.RequestEvent, l *core.Record) error {
	items, err := listItems(e.App, l.Id)
	if err != nil {
		return err
	}
	out := toList(e, l, true)
	out.Items = make([]itemJSON, 0, len(items))
	for _, it := range items {
		out.Items = append(out.Items, toItem(it)) // never includes reservations
	}
	out.Count = len(items)
	return e.JSON(http.StatusOK, out)
}

func patchList(e *core.RequestEvent, l *core.Record) error {
	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Preset      *string `json:"preset"`
	}
	if err := e.BindBody(&body); err != nil {
		return uerr(400, "invalid_body", "Invalid request.").api()
	}
	if body.Title != nil {
		l.Set("title", strings.TrimSpace(*body.Title))
	}
	if body.Description != nil {
		l.Set("description", strings.TrimSpace(*body.Description))
	}
	if body.Preset != nil {
		if !validPreset(*body.Preset) {
			return uerr(400, "invalid_body", "Unknown preset.").api()
		}
		l.Set("preset", *body.Preset)
	}
	if err := e.App.Save(l); err != nil {
		return saveErr(e, err)
	}
	return e.JSON(http.StatusOK, toList(e, l, true))
}

func deleteList(e *core.RequestEvent, l *core.Record) error {
	if err := e.App.Delete(l); err != nil {
		return err
	}
	return e.NoContent(http.StatusNoContent)
}

// dashboard returns the lists for the edit tokens this browser remembers plus,
// when signed in, every list the user owns. Unknown tokens come back in `missing`
// so the client can forget them.
func dashboard(e *core.RequestEvent) error {
	var body claimBody
	if err := e.BindBody(&body); err != nil {
		return uerr(400, "invalid_body", "Invalid request.").api()
	}
	if len(body.Tokens) > maxDashTokens {
		body.Tokens = body.Tokens[:maxDashTokens]
	}
	tokens := make([]any, 0, len(body.Tokens))
	for _, t := range body.Tokens {
		if len(t) >= 20 {
			tokens = append(tokens, t)
		}
	}

	var recs []*core.Record
	seen := map[string]bool{}
	add := func(rs []*core.Record) {
		for _, r := range rs {
			if !seen[r.Id] {
				seen[r.Id] = true
				recs = append(recs, r)
			}
		}
	}
	if len(tokens) > 0 {
		rs, err := e.App.FindAllRecords("lists", dbx.In("edit_token", tokens...))
		if err != nil {
			return err
		}
		add(rs)
	}
	if u := authUser(e); u != nil {
		rs, err := e.App.FindAllRecords("lists", dbx.HashExp{"owner": u.Id})
		if err != nil {
			return err
		}
		add(rs)
	}

	found := map[string]bool{}
	lists := make([]listJSON, 0, len(recs))
	for _, r := range recs {
		found[r.GetString("edit_token")] = true
		out := toList(e, r, true)
		n, _ := e.App.CountRecords("items", dbx.HashExp{"list": r.Id})
		out.Count = int(n)
		preview, _ := e.App.FindRecordsByFilter("items", "list = {:l}", "position,created", 4, 0, dbx.Params{"l": r.Id})
		out.Preview = []string{}
		for _, p := range preview {
			out.Preview = append(out.Preview, p.GetString("name"))
		}
		lists = append(lists, out)
	}
	slices.SortFunc(lists, func(a, b listJSON) int { return strings.Compare(b.Updated, a.Updated) })

	missing := []string{}
	for _, t := range tokens {
		if !found[t.(string)] {
			missing = append(missing, t.(string))
		}
	}
	return e.JSON(http.StatusOK, map[string]any{
		"lists": lists, "missing": missing,
		"gifts": gifts(e, body.Reservations), "visited": visited(e),
	})
}

// claimBody is what a browser knows without an account: edit tokens of its
// lists, reservation keys (item → key) and share ids it visited.
type claimBody struct {
	Tokens       []string          `json:"tokens"`
	Reservations map[string]string `json:"reservations"`
	Visits       []string          `json:"visits"`
}

// claim moves everything this browser knows into the signed-in account:
// ownerless lists, unowned reservations it holds the key for, and visited lists.
func claim(e *core.RequestEvent) error {
	var body claimBody
	if err := e.BindBody(&body); err != nil {
		return uerr(400, "invalid_body", "Invalid request.").api()
	}
	u := authUser(e)
	if u == nil {
		return uerr(401, "sign_in_first", "Sign in first.").api()
	}
	claimed, reservations, visits := 0, 0, 0
	for i, t := range body.Tokens {
		if i >= maxDashTokens || len(t) < 20 {
			continue
		}
		l, err := e.App.FindFirstRecordByData("lists", "edit_token", t)
		if err != nil || l.GetString("owner") != "" {
			continue
		}
		l.Set("owner", u.Id)
		if err := e.App.Save(l); err == nil {
			claimed++
		}
	}
	for _, r := range reservationsByKey(e.App, body.Reservations) {
		if r.GetString("user") == "" {
			r.Set("user", u.Id)
			if err := e.App.Save(r); err == nil {
				reservations++
			}
		}
	}
	for i, share := range body.Visits {
		if i >= maxDashTokens || len(share) < 8 {
			continue
		}
		l, err := e.App.FindFirstRecordByData("lists", "share_id", share)
		if err != nil || l.GetString("owner") == u.Id {
			continue
		}
		recordVisit(e.App, u.Id, l.Id)
		visits++
	}
	return e.JSON(http.StatusOK, map[string]any{"claimed": claimed, "reservations": reservations, "visits": visits})
}

// reservationsByKey returns the reservations whose item→key pairs match (a browser's proof).
func reservationsByKey(app core.App, keys map[string]string) []*core.Record {
	if len(keys) == 0 {
		return nil
	}
	ids := make([]any, 0, len(keys))
	for id := range keys {
		if len(ids) >= maxDashTokens {
			break
		}
		ids = append(ids, id)
	}
	rs, err := app.FindAllRecords("reservations", dbx.In("item", ids...))
	if err != nil {
		return nil
	}
	out := rs[:0]
	for _, r := range rs {
		if k := keys[r.GetString("item")]; k != "" && security.Equal(k, r.GetString("key")) {
			out = append(out, r)
		}
	}
	return out
}

func recordVisit(app core.App, userID, listID string) {
	v, err := app.FindFirstRecordByFilter("visits", "user = {:u} && list = {:l}", dbx.Params{"u": userID, "l": listID})
	if err != nil {
		col, err := app.FindCollectionByNameOrId("visits")
		if err != nil {
			return
		}
		v = core.NewRecord(col)
		v.Set("user", userID)
		v.Set("list", listID)
	}
	_ = app.Save(v) // bumps `updated` → most recent first
}

type giftJSON struct {
	Item   itemJSON `json:"item"`
	Title  string   `json:"title"`
	Preset string   `json:"preset"`
	Share  string   `json:"share"`
}

type visitJSON struct {
	Share  string `json:"share"`
	Title  string `json:"title"`
	Preset string `json:"preset"`
	At     string `json:"at"`
}

// gifts lists what this browser (keys) and/or account reserved, across all lists.
func gifts(e *core.RequestEvent, keys map[string]string) []giftJSON {
	recs := reservationsByKey(e.App, keys)
	if u := authUser(e); u != nil {
		mine, _ := e.App.FindAllRecords("reservations", dbx.HashExp{"user": u.Id})
		seen := map[string]bool{}
		for _, r := range recs {
			seen[r.Id] = true
		}
		for _, r := range mine {
			if !seen[r.Id] {
				recs = append(recs, r)
			}
		}
	}
	out := []giftJSON{}
	for _, r := range recs {
		it, err := e.App.FindRecordById("items", r.GetString("item"))
		if err != nil {
			continue
		}
		l, err := e.App.FindRecordById("lists", it.GetString("list"))
		if err != nil {
			continue
		}
		out = append(out, giftJSON{Item: toItem(it), Title: l.GetString("title"), Preset: l.GetString("preset"), Share: l.GetString("share_id")})
	}
	return out
}

func visited(e *core.RequestEvent) []visitJSON {
	out := []visitJSON{}
	u := authUser(e)
	if u == nil {
		return out
	}
	vs, err := e.App.FindRecordsByFilter("visits", "user = {:u}", "-updated", 50, 0, dbx.Params{"u": u.Id})
	if err != nil {
		return out
	}
	for _, v := range vs {
		l, err := e.App.FindRecordById("lists", v.GetString("list"))
		if err != nil || l.GetString("owner") == u.Id {
			continue
		}
		out = append(out, visitJSON{Share: l.GetString("share_id"), Title: l.GetString("title"), Preset: l.GetString("preset"), At: v.GetDateTime("updated").String()})
	}
	return out
}

// ---------- items ------------------------------------------------------------

type itemForm struct {
	name, description *string
	links             []string
	hasLinks          bool
	// order lists the final photos: "k:<existing name>", "f:<index into files>" or
	// "u:<image or page URL>" (downloaded server-side). nil = keep existing, append new.
	order          []string
	files          []*filesystem.File
	remote         map[string]*filesystem.File
	clearLinkPhoto bool
}

func readItemForm(e *core.RequestEvent) (*itemForm, error) {
	if err := e.Request.ParseMultipartForm(32 << 20); err != nil {
		return nil, uerr(400, "invalid_body", "Expected multipart/form-data.")
	}
	f := &itemForm{remote: map[string]*filesystem.File{}}
	form := e.Request.MultipartForm.Value
	if v, ok := form["name"]; ok && len(v) > 0 {
		s := strings.TrimSpace(v[0])
		f.name = &s
	}
	if v, ok := form["description"]; ok && len(v) > 0 {
		s := strings.TrimSpace(v[0])
		f.description = &s
	}
	if v, ok := form["links"]; ok && len(v) > 0 {
		var raw []string
		if err := json.Unmarshal([]byte(v[0]), &raw); err != nil {
			return nil, uerr(400, "invalid_body", "Links must be a JSON array.")
		}
		links, err := cleanLinks(raw)
		if err != nil {
			return nil, err
		}
		f.links, f.hasLinks = links, true
	}
	if v, ok := form["order"]; ok && len(v) > 0 {
		f.order = []string{}
		if err := json.Unmarshal([]byte(v[0]), &f.order); err != nil {
			return nil, uerr(400, "invalid_body", "Order must be a JSON array.")
		}
	}
	f.clearLinkPhoto = len(form["clear_link_photo"]) > 0 && form["clear_link_photo"][0] == "1"
	files, err := e.FindUploadedFiles("photos")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		return nil, err
	}
	f.files = files

	// photos added by URL: fetch them now so a bad URL is reported on save
	n := 0
	for _, o := range f.order {
		raw, ok := strings.CutPrefix(o, "u:")
		if !ok {
			continue
		}
		if n++; n > 8 {
			return nil, uerr(400, "too_many_photos", "Too many photos.", "max", 8)
		}
		links, err := cleanLinks([]string{raw})
		if err != nil || len(links) == 0 {
			return nil, uerr(400, "invalid_link", "Invalid image URL.", "url", raw)
		}
		file, _, err := resolveImage(e.Request.Context(), links[0])
		if err != nil {
			e.App.Logger().Debug("photo url failed", "url", links[0], "error", err)
			return nil, uerr(400, "image_fetch_failed", "Couldn’t get an image from "+hostOf(links[0])+" — try a direct image link, or upload it instead.", "host", hostOf(links[0]))
		}
		f.remote[raw] = file
	}
	return f, nil
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Host
	}
	return raw
}

func (f *itemForm) apply(it *core.Record) {
	if f.name != nil {
		it.Set("name", *f.name)
	}
	if f.description != nil {
		it.Set("description", *f.description)
	}
	if f.hasLinks {
		it.Set("links", f.links)
	}
	if f.clearLinkPhoto {
		it.Set("link_photo", nil) // link_photo_src stays, so it isn't fetched again for the same links
	}
	existing := it.GetStringSlice("photos")
	photos := []any{}
	if f.order == nil {
		for _, p := range existing {
			photos = append(photos, p)
		}
		for _, file := range f.files {
			photos = append(photos, file)
		}
	} else {
		usedFiles := map[int]bool{}
		for _, o := range f.order {
			kind, val, _ := strings.Cut(o, ":")
			switch kind {
			case "k":
				if slices.Contains(existing, val) {
					photos = append(photos, val)
				}
			case "f":
				if i, err := strconv.Atoi(val); err == nil && i >= 0 && i < len(f.files) && !usedFiles[i] {
					usedFiles[i] = true
					photos = append(photos, f.files[i])
				}
			case "u":
				if file := f.remote[val]; file != nil {
					photos = append(photos, file)
				}
			}
		}
	}
	it.Set("photos", photos) // dropped names get deleted from storage by PocketBase
}

func createItem(e *core.RequestEvent, l *core.Record) error {
	f, err := readItemForm(e)
	if err != nil {
		return apiError(err)
	}
	items, err := listItems(e.App, l.Id)
	if err != nil {
		return err
	}
	if len(items) >= maxItemsPerList {
		return uerr(400, "list_full", "This list is full.", "max", maxItemsPerList).api()
	}
	pos := 0
	if n := len(items); n > 0 {
		pos = items[n-1].GetInt("position") + 1
	}
	col, err := e.App.FindCollectionByNameOrId("items")
	if err != nil {
		return err
	}
	it := core.NewRecord(col)
	it.Set("list", l.Id)
	it.Set("position", pos)
	it.Set("links", []string{})
	f.apply(it)
	if err := e.App.Save(it); err != nil {
		return saveErr(e, err)
	}
	touch(e.App, l)
	return e.JSON(http.StatusCreated, toItem(it))
}

func updateItem(e *core.RequestEvent, l *core.Record) error {
	it, err := itemOf(e, l)
	if err != nil {
		return err
	}
	f, err := readItemForm(e)
	if err != nil {
		return apiError(err)
	}
	f.apply(it)
	if err := e.App.Save(it); err != nil {
		return saveErr(e, err)
	}
	touch(e.App, l)
	return e.JSON(http.StatusOK, toItem(it))
}

func deleteItem(e *core.RequestEvent, l *core.Record) error {
	it, err := itemOf(e, l)
	if err != nil {
		return err
	}
	if err := e.App.Delete(it); err != nil {
		return err
	}
	touch(e.App, l)
	return e.NoContent(http.StatusNoContent)
}

// reorder takes the full item id order (most wanted first) and renumbers positions.
func reorder(e *core.RequestEvent, l *core.Record) error {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := e.BindBody(&body); err != nil {
		return uerr(400, "invalid_body", "Invalid request.").api()
	}
	err := e.App.RunInTransaction(func(tx core.App) error {
		items, err := listItems(tx, l.Id)
		if err != nil {
			return err
		}
		byID := map[string]*core.Record{}
		for _, it := range items {
			byID[it.Id] = it
		}
		pos := 0
		place := func(it *core.Record) error {
			defer func() { pos++ }()
			if it.GetInt("position") == pos {
				return nil
			}
			it.Set("position", pos)
			return tx.Save(it)
		}
		for _, id := range body.IDs {
			if it := byID[id]; it != nil {
				delete(byID, id)
				if err := place(it); err != nil {
					return err
				}
			}
		}
		for _, it := range items { // anything the client didn't mention keeps its relative order at the end
			if byID[it.Id] != nil {
				if err := place(it); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	touch(e.App, l)
	return e.NoContent(http.StatusNoContent)
}

// ---------- share view & reservations ------------------------------------------

func listByShare(e *core.RequestEvent) (*core.Record, error) {
	id := e.Request.PathValue("share")
	if len(id) < 8 {
		return nil, uerr(404, "list_not_found", "List not found.").api()
	}
	l, err := e.App.FindFirstRecordByData("lists", "share_id", id)
	if err != nil {
		return nil, uerr(404, "list_not_found", "List not found.").api()
	}
	return l, nil
}

func getShareList(e *core.RequestEvent) error {
	l, err := listByShare(e)
	if err != nil {
		return err
	}
	items, err := listItems(e.App, l.Id)
	if err != nil {
		return err
	}
	reserved := map[string]bool{}
	byMe := map[string]bool{}
	u := authUser(e)
	if len(items) > 0 {
		ids := make([]any, len(items))
		for i, it := range items {
			ids[i] = it.Id
		}
		rs, err := e.App.FindAllRecords("reservations", dbx.In("item", ids...))
		if err != nil {
			return err
		}
		for _, r := range rs {
			reserved[r.GetString("item")] = true
			if u != nil && r.GetString("user") == u.Id {
				byMe[r.GetString("item")] = true
			}
		}
	}
	out := toList(e, l, false)
	if u != nil && !out.Mine {
		recordVisit(e.App, u.Id, l.Id)
	}
	out.Items = make([]itemJSON, 0, len(items))
	for _, it := range items {
		j := toItem(it)
		r := reserved[it.Id]
		j.Reserved = &r
		j.ReservedByMe = byMe[it.Id]
		out.Items = append(out.Items, j)
	}
	out.Count = len(items)
	return e.JSON(http.StatusOK, out)
}

func shareItem(e *core.RequestEvent) (*core.Record, error) {
	l, err := listByShare(e)
	if err != nil {
		return nil, err
	}
	return itemOf(e, l)
}

func reserve(e *core.RequestEvent) error {
	it, err := shareItem(e)
	if err != nil {
		return err
	}
	col, err := e.App.FindCollectionByNameOrId("reservations")
	if err != nil {
		return err
	}
	r := core.NewRecord(col)
	r.Set("item", it.Id)
	key := security.RandomString(32)
	r.Set("key", key)
	if u := authUser(e); u != nil {
		r.Set("user", u.Id)
	}
	if err := e.App.Save(r); err != nil {
		// unique index on item → someone was faster
		return uerr(409, "already_reserved", "Someone already reserved this.").api()
	}
	return e.JSON(http.StatusCreated, map[string]string{"key": key})
}

func unreserve(e *core.RequestEvent) error {
	it, err := shareItem(e)
	if err != nil {
		return err
	}
	key := e.Request.Header.Get("X-Reservation-Key")
	r, err := e.App.FindFirstRecordByData("reservations", "item", it.Id)
	if err != nil {
		return e.NoContent(http.StatusNoContent) // nothing to undo
	}
	u := authUser(e)
	byKey := key != "" && security.Equal(key, r.GetString("key"))
	byUser := u != nil && r.GetString("user") == u.Id
	if !byKey && !byUser {
		return uerr(403, "not_your_reservation", "Only the person who reserved this can release it.").api()
	}
	if err := e.App.Delete(r); err != nil {
		return err
	}
	return e.NoContent(http.StatusNoContent)
}

// linkPreview fetches a picture (and optionally a title) from the item's links.
// It's a separate call so saving stays fast; the client fires it after a save.
// Links that were already tried are not fetched again.
func linkPreview(e *core.RequestEvent, l *core.Record) error {
	it, err := itemOf(e, l)
	if err != nil {
		return err
	}
	var body struct {
		Autoname bool `json:"autoname"` // replace the name with the page title (quick-added links)
	}
	_ = e.BindBody(&body)

	links := []string{}
	_ = it.UnmarshalJSONField("links", &links)
	src := strings.Join(links, "\n")
	if src == it.GetString("link_photo_src") && !body.Autoname {
		return e.JSON(http.StatusOK, toItem(it))
	}
	it.Set("link_photo_src", src)
	it.Set("link_photo", nil) // links changed: drop the old picture unless we find a new one
	nameSet := false
	for i, link := range links {
		if i >= 3 {
			break
		}
		file, title, err := resolveImage(e.Request.Context(), link)
		if body.Autoname && !nameSet && title != "" {
			if r := []rune(title); len(r) > 200 {
				title = string(r[:200])
			}
			it.Set("name", title)
			nameSet = true
		}
		if err != nil {
			e.App.Logger().Debug("link preview failed", "url", link, "error", err)
			continue
		}
		it.Set("link_photo", file)
		break
	}
	if err := e.App.Save(it); err != nil {
		return saveErr(e, err)
	}
	return e.JSON(http.StatusOK, toItem(it))
}
