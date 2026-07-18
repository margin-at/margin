package api

import (
	"reflect"
	"testing"
)

func TestParseMotivationsIncludesTaggedAnnotations(t *testing.T) {
	want := []string{"commenting", "tagging"}
	if got := parseMotivations("commenting"); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseMotivations(commenting) = %v, want %v", got, want)
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
