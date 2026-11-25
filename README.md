# wishlist

Simple wishlists at **wish.fhun.site**. No account is needed: creating a list gives you two links.

- **Edit link** `/e/{token}` is private and is the key to the list. Use it to add wishes (name, details, photos, links) and drag them into priority order.
- **Share link** `/l/{id}` is for guests. They can see the wishes and press *I'll get this* to reserve one. The reservation key is stored only in their browser, so they can undo it on that device.

Photos can be uploaded, pasted, dragged in (files, or images dragged from another tab), or added by URL.
A URL can point to the image itself or to a product page. If a wish has no photo of its own, the picture from its links is used instead (`og:image`, Twitter cards, JSON-LD, Amazon's product image).
Pasting a product link into "Add a wish…" fills in the page title as well.
These fetches run on the server (`fetch.go`), so guests load the picture from this server rather than from the shop. Private, loopback and link-local addresses are refused after DNS resolution.
Shops with bot protection (some Amazon pages, Etsy, Thalia, …) can't be fetched this way; adding the photo by paste or drag-and-drop still works.

The list owner never sees reservations. The edit API doesn't return them, and the share view hides them when it recognizes the owner.
Signing in is optional. It moves this browser's lists into the account, so their edit links can't be lost.

Single Go binary (PocketBase + embedded Svelte 5 PWA).

## Routes

| Path | Page |
|---|---|
| `/` | landing / marketing |
| `/app` | dashboard (templates row + your lists, PWA start url) |
| `/e/{token}` | editor |
| `/l/{share}` | public list |
| `/_/` | PocketBase admin |

API lives under `/api/wl/` (see `api.go`). The `lists` and `reservations` collections are superuser-only. Clients reach them only through the capability routes.

## Presets

Background images are in `ui/public/presets/{blank,christmas,birthday,wedding,baby}.svg` (placeholders).
To use real images, drop them in that folder (e.g. `christmas.jpg`) and update `image` and the `accent` colour in `ui/src/lib/presets.ts`.

## Languages

English and German are built in (`ui/src/lib/i18n/`). The app follows the browser/OS language by default, and visitors can switch in the top bar (Automatic / English / Deutsch). Their choice is remembered in the browser.

To add a language:

1. Copy `en.ts` to e.g. `fr.ts`, write `const fr: Messages = { … }; export default fr`, and translate the values.
2. Add one line to `LANGUAGES` in `i18n/index.svelte.ts`: `{ code: 'fr', name: 'Français', messages: fr }`.
3. Run `npm run check`. It fails on any missing or misspelled key.

Message syntax is plain text, never HTML: `{param}`, `**bold**`, `_emphasis_`, `[the one link/button in a sentence]`, and plural objects `{ one: '…', other: '…' }` (selected by `count` via `Intl.PluralRules`).
API errors include a stable `code` (see `errors.go`), translated under `errors.*`; an unknown code falls back to the server's English message.

## Dev

```sh
make dev               # Go API on :8080
cd ui && npm run dev   # Vite on :5173, proxies /api
make build             # ./wishlist
docker compose up -d --build
```

Rate limiting is enabled and uses traefik's `X-Real-IP` header (set in the init migration). If you expose the binary without a proxy, remove that trusted header in `/_/ → Settings`.
