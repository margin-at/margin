package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestParseMotivationsIncludesTaggedAnnotations(t *testing.T) {
	want := []string{"commenting", "tagging"}
	if got := parseMotivations("commenting"); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseMotivations(commenting) = %v, want %v", got, want)
	}
}

func TestFetchURLMetadataExtractsAtCanonical(t *testing.T) {
	page := `<!DOCTYPE html>
<html>
<head>
<title>Test Page</title>
<meta property="og:title" content="OG Title">
<meta name="at:canonical" content="at://did:plc:abc123/app.bsky.feed.post/xyz">
<meta name="at:canonical" content="at://did:plc:abc123/app.bsky.feed.post/xyz">
<meta name='at:canonical' content='at://did:plc:def456/at.margin.note/rkey1'>
<meta name="at:canonical" content="not-an-at-uri">
</head>
<body></body>
</html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(page))
	}))
	defer server.Close()

	h := &Handler{}
	data := h.fetchURLMetadata(context.Background(), server.URL)

	if data["title"] != "OG Title" {
		t.Fatalf("title = %q, want %q", data["title"], "OG Title")
	}
	want := "at://did:plc:abc123/app.bsky.feed.post/xyz at://did:plc:def456/at.margin.note/rkey1"
	if data["at:canonical"] != want {
		t.Fatalf("at:canonical = %q, want %q", data["at:canonical"], want)
	}
}

func TestFetchURLMetadataOmitsAtCanonicalWhenAbsent(t *testing.T) {
	page := `<html><head><title>Plain</title></head><body></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(page))
	}))
	defer server.Close()

	h := &Handler{}
	data := h.fetchURLMetadata(context.Background(), server.URL)

	if data["title"] != "Plain" {
		t.Fatalf("title = %q, want %q", data["title"], "Plain")
	}
	if v, ok := data["at:canonical"]; ok {
		t.Fatalf("at:canonical should be absent, got %q", v)
	}
}

func TestParseMotivationsPreservesOtherFilters(t *testing.T) {
	for _, motivation := range []string{"highlighting", "bookmarking", ""} {
		t.Run(motivation, func(t *testing.T) {
			var want []string
			if motivation != "" {
				want = []string{motivation}
			}
			if got := parseMotivations(motivation); !reflect.DeepEqual(got, want) {
				t.Fatalf("parseMotivations(%q) = %v, want %v", motivation, got, want)
			}
		})
	}
}
