package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestExtractPreview(t *testing.T) {
	base, _ := url.Parse("https://shop.example/p/1")
	cases := []struct{ name, page, img, title string }{
		{"og", `<meta property="og:title" content="Chef&#39;s Knife"><meta name="twitter:image" content="/tw.jpg"><meta property="og:image" content="https://cdn.example/a.jpg?x=1&amp;y=2">`,
			"https://cdn.example/a.jpg?x=1&y=2", "Chef's Knife"},
		{"relative twitter", `<title> Mug  |  Shop </title><meta content='/img/m.png' name='twitter:image'>`, "https://shop.example/img/m.png", "Mug | Shop"},
		{"jsonld", `<script type="application/ld+json">{"@type":"Product","image":["https:\/\/cdn.example\/j.webp"]}</script>`, "https://cdn.example/j.webp", ""},
		{"amazon", `<img alt="x" id="landingImage" src="https://m.media/s.jpg" data-old-hires="https://m.media/big.jpg">`, "https://m.media/big.jpg", ""},
		{"none", `<p>hi</p>`, "", ""},
	}
	for _, c := range cases {
		img, title := extractPreview(c.page, base)
		if img != c.img || title != c.title {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", c.name, img, title, c.img, c.title)
		}
	}
}

func TestFetchBlocksPrivateAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("secret")) }))
	defer srv.Close()
	_, err := httpGet(context.Background(), srv.URL, "*/*")
	if !errors.Is(err, errBlockedAddr) {
		t.Fatalf("expected loopback to be blocked, got %v", err)
	}
	if _, err := httpGet(context.Background(), "file:///etc/passwd", "*/*"); err == nil {
		t.Fatal("expected non-http scheme to be rejected")
	}
}
