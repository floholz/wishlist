package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// Outbound fetching of user-supplied URLs (product pages, image URLs).
// The dialer refuses non-public addresses *after* DNS resolution, so neither
// "http://localhost" nor a DNS name pointing at 10.x can reach internal services.

const (
	maxPageBytes  = 1536 << 10
	maxImageBytes = 10 << 20
	userAgent     = "Mozilla/5.0 (compatible; wishlist-linkpreview/1.0; +https://wish.fhun.site)"
)

var errBlockedAddr = errors.New("address not allowed")

var fetchClient = &http.Client{
	Timeout: 12 * time.Second,
	Transport: &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
			Control: func(_, address string, _ syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					return err
				}
				ip := net.ParseIP(host)
				if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
					return errBlockedAddr
				}
				if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1]&0xc0 == 64 { // 100.64/10 CGNAT
					return errBlockedAddr
				}
				return nil
			},
		}).DialContext,
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  8 * time.Second,
		MaxResponseHeaderBytes: 64 << 10,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return errors.New("bad redirect scheme")
		}
		return nil
	},
}

func httpGet(ctx context.Context, rawURL, accept string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("invalid URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)
	req.Header.Set("Accept-Language", "en,de;q=0.8")
	res, err := fetchClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return res, nil
}

var imageExt = map[string]string{
	"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif", "image/avif": ".avif",
}

// readImage turns a response body into an upload, sniffing the real type (SVG and friends are rejected).
func readImage(res *http.Response) (*filesystem.File, error) {
	b, err := io.ReadAll(io.LimitReader(res.Body, maxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxImageBytes {
		return nil, errors.New("image is too large")
	}
	ct := http.DetectContentType(b)
	if strings.HasPrefix(ct, "application/octet-stream") || strings.HasPrefix(ct, "text/") { // avif isn't sniffed
		if h, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type")); h == "image/avif" {
			ct = h
		}
	}
	ext, ok := imageExt[strings.SplitN(ct, ";", 2)[0]]
	if !ok {
		return nil, errors.New("not a supported image")
	}
	return filesystem.NewFileFromBytes(b, "web"+ext)
}

// resolveImage accepts either a direct image URL or a web page; for pages it
// picks the page's preview image. Also returns the page title (if any).
func resolveImage(ctx context.Context, rawURL string) (*filesystem.File, string, error) {
	res, err := httpGet(ctx, rawURL, "text/html,application/xhtml+xml,image/*;q=0.9,*/*;q=0.5")
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	ct, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if strings.HasPrefix(ct, "image/") {
		f, err := readImage(res)
		return f, "", err
	}
	if ct != "" && ct != "text/html" && ct != "application/xhtml+xml" {
		return nil, "", errors.New("not a web page or image")
	}
	page, err := io.ReadAll(io.LimitReader(res.Body, maxPageBytes))
	if err != nil {
		return nil, "", err
	}
	imgURL, title := extractPreview(string(page), res.Request.URL)
	if imgURL == "" {
		return nil, title, errors.New("no preview image found")
	}
	ires, err := httpGet(ctx, imgURL, "image/avif,image/webp,image/*;q=0.9")
	if err != nil {
		return nil, title, err
	}
	defer ires.Body.Close()
	f, err := readImage(ires)
	return f, title, err
}

var (
	reTag      = regexp.MustCompile(`(?is)<(meta|link|img)\b[^>]*>`)
	reAttr     = regexp.MustCompile(`(?is)([a-z_:][-a-z0-9_:.]*)\s*=\s*("[^"]*"|'[^']*'|[^\s"'>]+)`)
	reTitle    = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reJSONLDIm = regexp.MustCompile(`(?is)"image"\s*:\s*(?:\[\s*)?(?:\{[^}]*?"url"\s*:\s*)?"(https?:[^"]+)"`)
	reHiRes    = regexp.MustCompile(`"hiRes"\s*:\s*"(https?:[^"]+)"`)
)

func attrs(tag string) map[string]string {
	out := map[string]string{}
	for _, m := range reAttr.FindAllStringSubmatch(tag, -1) {
		v := strings.Trim(m[2], `"'`)
		out[strings.ToLower(m[1])] = html.UnescapeString(strings.TrimSpace(v))
	}
	return out
}

// extractPreview finds the best "product image" of a page: OpenGraph / Twitter
// cards first, then JSON-LD, then a few well-known shop markups.
func extractPreview(page string, base *url.URL) (img, title string) {
	rank := map[string]int{ // lower = better
		"og:image:secure_url": 1, "og:image": 2, "og:image:url": 2, "twitter:image": 3, "twitter:image:src": 3,
		"image_src": 4, "itemprop:image": 5, "shop": 6,
	}
	best, bestRank := "", 99
	consider := func(key, v string) {
		if r, ok := rank[key]; ok && v != "" && r < bestRank {
			best, bestRank = v, r
		}
	}
	for _, tag := range reTag.FindAllString(page, -1) {
		a := attrs(tag)
		switch strings.ToLower(tag[1:4]) {
		case "met":
			key := strings.ToLower(a["property"])
			if key == "" {
				key = strings.ToLower(a["name"])
			}
			if key == "og:title" && title == "" {
				title = a["content"]
			}
			consider(key, a["content"])
			if a["itemprop"] == "image" {
				consider("itemprop:image", a["content"])
			}
		case "lin":
			if strings.EqualFold(a["rel"], "image_src") {
				consider("image_src", a["href"])
			}
		case "img":
			if a["id"] == "landingImage" { // Amazon
				if v := a["data-old-hires"]; v != "" {
					consider("shop", v)
				} else {
					consider("shop", a["src"])
				}
			} else if a["itemprop"] == "image" {
				consider("itemprop:image", firstNonEmpty(a["src"], a["data-src"]))
			}
		}
	}
	if bestRank > 4 {
		if m := reJSONLDIm.FindStringSubmatch(page); m != nil {
			consider("itemprop:image", strings.ReplaceAll(m[1], `\/`, `/`))
			bestRank = min(bestRank, 4)
		}
	}
	if best == "" {
		if m := reHiRes.FindStringSubmatch(page); m != nil {
			best = m[1]
		}
	}
	if title == "" {
		if m := reTitle.FindStringSubmatch(page); m != nil {
			title = html.UnescapeString(strings.Join(strings.Fields(m[1]), " "))
		}
	}
	if best != "" && !strings.HasPrefix(best, "data:") {
		if u, err := base.Parse(best); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			img = u.String()
		}
	}
	return img, strings.TrimSpace(title)
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
