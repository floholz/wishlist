package main

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// configureGoogle enables "Sign in with Google" on the users collection from
// WISHLIST_GOOGLE_CLIENT_ID / WISHLIST_GOOGLE_CLIENT_SECRET. Without them,
// whatever is configured in the admin UI (/_/) is left alone.
func configureGoogle(app core.App) error {
	id, secret := env("GOOGLE_CLIENT_ID"), env("GOOGLE_CLIENT_SECRET")
	if id == "" || secret == "" {
		return nil
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	found := false
	for i, p := range users.OAuth2.Providers {
		if p.Name == "google" {
			if p.ClientId == id && p.ClientSecret == secret && users.OAuth2.Enabled {
				return nil // already up to date
			}
			users.OAuth2.Providers[i].ClientId = id
			users.OAuth2.Providers[i].ClientSecret = secret
			found = true
		}
	}
	if !found {
		users.OAuth2.Providers = append(users.OAuth2.Providers, core.OAuth2ProviderConfig{
			Name: "google", ClientId: id, ClientSecret: secret,
		})
	}
	users.OAuth2.Enabled = true
	if users.Fields.GetByName("name") != nil && users.OAuth2.MappedFields.Name == "" {
		users.OAuth2.MappedFields.Name = "name"
	}
	if users.Fields.GetByName("avatar") != nil && users.OAuth2.MappedFields.AvatarURL == "" {
		users.OAuth2.MappedFields.AvatarURL = "avatar"
	}
	if err := app.Save(users); err != nil {
		return err
	}
	app.Logger().Info("configured Google sign-in from env")
	return nil
}

// fillMissingAvatar gives an existing account (e.g. one made with email and
// password, later signed into with Google) the provider's picture. New accounts
// get it from PocketBase's field mapping already. It runs right before the auth
// response is sent, so the browser gets the record with the picture.
func fillMissingAvatar(e *core.RecordAuthRequestEvent) {
	meta, _ := e.Meta.(map[string]any)
	url, _ := meta["avatarURL"].(string)
	if e.AuthMethod != core.MFAMethodOAuth2 || url == "" || e.Record == nil ||
		e.Record.Collection().Fields.GetByName("avatar") == nil || e.Record.GetString("avatar") != "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	f, err := filesystem.NewFileFromURL(ctx, url)
	if err != nil {
		e.App.Logger().Warn("failed to fetch OAuth2 avatar", "error", err)
		return
	}
	e.Record.Set("avatar", f)
	if err := e.App.Save(e.Record); err != nil {
		e.App.Logger().Warn("failed to save OAuth2 avatar", "error", err)
	}
}
