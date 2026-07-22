package postgres

import (
	"testing"
	"time"
)

type contentLabelRow struct {
	neg int32
}

func (r contentLabelRow) Scan(dest ...interface{}) error {
	createdAt := time.Date(2026, time.July, 21, 20, 17, 0, 0, time.UTC)
	*dest[0].(*int32) = 42
	*dest[1].(*string) = "did:plc:labeler"
	*dest[2].(*string) = "at://did:plc:author/at.margin.note/example"
	*dest[3].(*string) = "gore"
	*dest[4].(*int32) = r.neg
	*dest[5].(*string) = "did:plc:moderator"
	*dest[6].(*time.Time) = createdAt
	return nil
}

func TestScanContentLabelConvertsIntegerNegation(t *testing.T) {
	for _, test := range []struct {
		name string
		neg  int32
		want bool
	}{
		{name: "active", neg: 0, want: false},
		{name: "negated", neg: 1, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			label, err := scanContentLabel(contentLabelRow{neg: test.neg})
			if err != nil {
				t.Fatalf("scanContentLabel() error = %v", err)
			}
			if label.Neg != test.want {
				t.Fatalf("scanContentLabel().Neg = %v, want %v", label.Neg, test.want)
			}
			if label.ID != 42 || label.URI != "at://did:plc:author/at.margin.note/example" {
				t.Fatalf("scanContentLabel() = %+v", label)
			}
		})
	}
}
