package xrpc

import "strings"

const (
	CollectionLichenBookmark = "wiki.lichen.bookmark"
	CollectionLichenWiki     = "wiki.lichen.wiki"
)

type LichenBookmark struct {
	WikiRef   string `json:"wikiRef"`
	CreatedAt string `json:"createdAt"`
}

func LichenWikiURLFromRef(wikiRef string) string {
	rest := strings.TrimPrefix(wikiRef, "at://")
	parts := strings.Split(rest, "/")
	if len(parts) < 3 || parts[1] != CollectionLichenWiki {
		return ""
	}
	rkey := parts[len(parts)-1]
	if rkey == "" {
		return ""
	}
	return "https://lichen.wiki/wiki/" + rkey
}
