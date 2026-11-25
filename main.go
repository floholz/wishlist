// wishlist — wishlists without accounts. One edit link, one share link.
package main

import (
	"embed"
	"io/fs"
	"log"
	"mime"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/cmd"
	"github.com/pocketbase/pocketbase/core"

	_ "github.com/floholz/wishlist/migrations"
)

//go:embed all:ui/dist
var uiEmbed embed.FS

const defaultAddr = "127.0.0.1:8080"

// version is set at build time: -ldflags "-X main.version=v1.0.0"
var version = "dev"

// env reads WISHLIST_<name>, falling back to the old WHISH_<name>.
func env(name string) string {
	if v, ok := os.LookupEnv("WISHLIST_" + name); ok {
		return v
	}
	return os.Getenv("WHISH_" + name)
}

// Registration policy (WISHLIST_REGISTRATION): unset / "true" → open, "false" → closed.
func registrationOpen() bool {
	switch strings.ToLower(env("REGISTRATION")) {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

func main() {
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
	app := pocketbase.New()

	app.OnRecordCreateRequest("users").BindFunc(func(e *core.RecordRequestEvent) error {
		if !e.HasSuperuserAuth() && !registrationOpen() {
			return uerr(403, "registration_closed", "Registration is closed on this instance.").api()
		}
		return e.Next()
	})

	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := bootstrapSuperuser(app); err != nil {
			return err
		}
		ui, err := fs.Sub(uiEmbed, "ui/dist")
		if err != nil {
			return err
		}
		registerAPI(e)
		e.Router.GET("/api/wl/status", func(re *core.RequestEvent) error {
			return re.JSON(200, map[string]any{"registration": registrationOpen(), "version": version})
		})
		e.Router.GET("/{path...}", apis.Static(ui, true)) // SPA: unknown paths fall back to index.html
		return e.Next()
	})

	// Same as app.Start(), but with our own default listen address.
	app.RootCmd.AddCommand(cmd.NewSuperuserCommand(app))
	serve := cmd.NewServeCommand(app, true)
	if f := serve.PersistentFlags().Lookup("http"); f != nil {
		_ = f.Value.Set(defaultAddr)
		f.DefValue = defaultAddr
		f.Usage = "TCP address to listen for the HTTP server"
	}
	app.RootCmd.AddCommand(serve)
	if err := app.Execute(); err != nil {
		log.Fatal(err)
	}
}

// bootstrapSuperuser creates the PocketBase admin (for /_/) from
// WISHLIST_ADMIN_EMAIL / WISHLIST_ADMIN_PASSWORD when no superuser exists yet.
func bootstrapSuperuser(app core.App) error {
	email, pass := env("ADMIN_EMAIL"), env("ADMIN_PASSWORD")
	if email == "" || pass == "" {
		return nil
	}
	n, err := app.CountRecords(core.CollectionNameSuperusers)
	if err != nil || n > 0 {
		return err
	}
	col, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return err
	}
	u := core.NewRecord(col)
	u.SetEmail(email)
	u.SetPassword(pass)
	if err := app.Save(u); err != nil {
		return err
	}
	app.Logger().Info("created superuser from env", "email", email)
	return nil
}
