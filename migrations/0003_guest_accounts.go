package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// v3: guests can sign in too.
//
//	reservations.user — the account that reserved (optional; the browser key keeps working)
//	visits            — share links a signed-in user opened, so they can find them again anywhere
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		res, err := app.FindCollectionByNameOrId("reservations")
		if err != nil {
			return err
		}
		if res.Fields.GetByName("user") == nil {
			res.Fields.Add(&core.RelationField{Name: "user", MaxSelect: 1, CollectionId: users.Id})
			res.AddIndex("idx_reservations_user", false, "user", "")
			if err := app.Save(res); err != nil {
				return err
			}
		}

		lists, err := app.FindCollectionByNameOrId("lists")
		if err != nil {
			return err
		}
		visits := core.NewBaseCollection("visits")
		visits.Fields.Add(
			&core.RelationField{Name: "user", Required: true, MaxSelect: 1, CollectionId: users.Id, CascadeDelete: true},
			&core.RelationField{Name: "list", Required: true, MaxSelect: 1, CollectionId: lists.Id, CascadeDelete: true},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		visits.AddIndex("idx_visits_user_list", true, "user, list", "")
		return app.Save(visits) // superuser-only, like lists & reservations
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId("visits"); err == nil {
			if err := app.Delete(c); err != nil {
				return err
			}
		}
		res, err := app.FindCollectionByNameOrId("reservations")
		if err != nil {
			return err
		}
		res.RemoveIndex("idx_reservations_user")
		res.Fields.RemoveByName("user")
		return app.Save(res)
	})
}
