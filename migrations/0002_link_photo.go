package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// v2: a fallback picture fetched from the item's links (og:image & co).
//
//	link_photo     — the downloaded preview image (only shown when the item has no own photos)
//	link_photo_src — the links it was derived from, so we only refetch when they change
func init() {
	m.Register(func(app core.App) error {
		items, err := app.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		if items.Fields.GetByName("link_photo") == nil {
			items.Fields.Add(&core.FileField{
				Name: "link_photo", MaxSelect: 1, MaxSize: 10 << 20,
				MimeTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif", "image/avif"},
				Thumbs:    []string{"480x480", "1200x0"},
			})
		}
		if items.Fields.GetByName("link_photo_src") == nil {
			items.Fields.Add(&core.TextField{Name: "link_photo_src", Max: 25000, Hidden: true})
		}
		if err := app.Save(items); err != nil {
			return err
		}
		// every item save may trigger outbound fetches — keep that bounded per client
		s := app.Settings()
		s.RateLimits.Rules = append([]core.RateLimitRule{
			{Label: "POST /api/wl/e/", Duration: 60, MaxRequests: 60},
			{Label: "PATCH /api/wl/e/", Duration: 60, MaxRequests: 60},
		}, s.RateLimits.Rules...)
		return app.Save(s)
	}, func(app core.App) error {
		items, err := app.FindCollectionByNameOrId("items")
		if err != nil {
			return err
		}
		items.Fields.RemoveByName("link_photo")
		items.Fields.RemoveByName("link_photo_src")
		return app.Save(items)
	})
}
