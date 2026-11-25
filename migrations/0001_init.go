package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// v1 schema.
//
//	lists        — title, description, preset, share_id (public link), edit_token (secret link), owner (optional user)
//	items        — list, name, description, links, photos, position (priority, 0 = most wanted)
//	reservations — item (unique), key (secret held by the reserving browser)
//
// lists and reservations are superuser-only: every client access goes through the
// capability routes in api.go (/api/wl/…), which check the share id / edit token.
// items are viewable by id so PocketBase can serve their photos; nothing about
// reservations lives on an item, so the list owner can never see them.
//
// Never edit an applied migration — later changes live in their own numbered files.

var Presets = []string{"blank", "christmas", "birthday", "wedding", "baby"}

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}

		lists := core.NewBaseCollection("lists")
		lists.Fields.Add(
			&core.RelationField{Name: "owner", MaxSelect: 1, CollectionId: users.Id},
			&core.TextField{Name: "title", Required: true, Max: 120},
			&core.TextField{Name: "description", Max: 2000},
			&core.SelectField{Name: "preset", Required: true, MaxSelect: 1, Values: Presets},
			&core.TextField{Name: "share_id", Required: true, Min: 8, Max: 32},
			&core.TextField{Name: "edit_token", Required: true, Min: 20, Max: 64, Hidden: true},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		lists.AddIndex("idx_lists_share", true, "share_id", "")
		lists.AddIndex("idx_lists_edit", true, "edit_token", "")
		lists.AddIndex("idx_lists_owner", false, "owner", "")
		if err := app.Save(lists); err != nil {
			return err
		}

		items := core.NewBaseCollection("items")
		public := ""
		items.ViewRule = &public // photos are served by /api/files, which checks the view rule
		items.Fields.Add(
			&core.RelationField{Name: "list", Required: true, MaxSelect: 1, CollectionId: lists.Id, CascadeDelete: true},
			&core.TextField{Name: "name", Required: true, Max: 200},
			&core.TextField{Name: "description", Max: 4000},
			&core.JSONField{Name: "links", MaxSize: 32 << 10},
			&core.FileField{
				Name: "photos", MaxSelect: 8, MaxSize: 10 << 20,
				MimeTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif", "image/avif"},
				Thumbs:    []string{"480x480", "1200x0"},
			},
			&core.NumberField{Name: "position"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		items.AddIndex("idx_items_list", false, "list, position", "")
		if err := app.Save(items); err != nil {
			return err
		}

		res := core.NewBaseCollection("reservations")
		res.Fields.Add(
			&core.RelationField{Name: "item", Required: true, MaxSelect: 1, CollectionId: items.Id, CascadeDelete: true},
			&core.TextField{Name: "key", Required: true, Min: 20, Max: 64, Hidden: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		res.AddIndex("idx_reservations_item", true, "item", "") // one reservation per item, race-safe
		if err := app.Save(res); err != nil {
			return err
		}

		// Behind traefik every request comes from the proxy — trust its real-ip header
		// so the rate limiter sees actual clients.
		s := app.Settings()
		s.Meta.AppName = "whish"
		s.TrustedProxy.Headers = []string{"X-Real-IP"}
		s.RateLimits.Enabled = true
		// first matching prefix wins, so ours go before PocketBase's generic "/api/" rule
		s.RateLimits.Rules = append([]core.RateLimitRule{
			{Label: "POST /api/wl/lists", Duration: 600, MaxRequests: 20},
			{Label: "POST /api/wl/s/", Duration: 60, MaxRequests: 30},
			{Label: "/api/wl/", Duration: 60, MaxRequests: 300},
		}, s.RateLimits.Rules...)
		return app.Save(s)
	}, func(app core.App) error {
		for _, name := range []string{"reservations", "items", "lists"} {
			if c, err := app.FindCollectionByNameOrId(name); err == nil {
				if err := app.Delete(c); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
