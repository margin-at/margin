package db

import "testing"

func TestIsCollectionURI(t *testing.T) {
	for _, test := range []struct {
		name string
		uri  string
		want bool
	}{
		{
			name: "margin collection",
			uri:  "at://did:plc:example/at.margin.collection/3abc",
			want: true,
		},
		{
			name: "semble collection",
			uri:  "at://did:plc:example/network.cosmik.collection/3abc",
			want: true,
		},
		{
			name: "collection item",
			uri:  "at://did:plc:example/at.margin.collectionItem/3abc",
			want: false,
		},
		{
			name: "invalid URI",
			uri:  "not-an-at-uri",
			want: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isCollectionURI(test.uri); got != test.want {
				t.Fatalf("isCollectionURI(%q) = %v, want %v", test.uri, got, test.want)
			}
		})
	}
}
