package service

import "testing"

func TestNormalizeSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		raw       string
		want      string
		wantError bool
	}{
		{name: "valid lowercase", raw: "saksa-12a", want: "saksa-12a"},
		{name: "trim and lowercase", raw: "Lesi-8B ", want: "lesi-8b"},
		{name: "too short", raw: "ab", wantError: true},
		{name: "invalid chars", raw: "a b", wantError: true},
		{name: "reserved api lowercase", raw: "api", wantError: true},
		{name: "reserved api mixed case", raw: "ApI", wantError: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeSlug(tc.raw)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil (slug=%q)", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected slug: got %q want %q", got, tc.want)
			}
		})
	}
}
