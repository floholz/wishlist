package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// v4: the app is called "wishlist" (0001 set the misspelled "whish").
func init() {
	m.Register(func(app core.App) error {
		s := app.Settings()
		if s.Meta.AppName != "whish" {
			return nil // changed by hand in /_/ — leave it
		}
		s.Meta.AppName = "wishlist"
		return app.Save(s)
	}, nil)
}
