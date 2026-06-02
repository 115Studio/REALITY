package reality

import (
	"strings"
	"testing"
)

func TestMatchServerNameWildcard(t *testing.T) {
	c := &Config{ServerNames: map[string]bool{
		"de-fra-1-*.blook.network": true,
		"exact.example.com":        true,
	}}

	cases := []struct {
		name string
		want bool
	}{
		{"exact.example.com", true},                                       // exact entry
		{"de-fra-1-abc123.blook.network", true},                           // typical wildcard fill
		{"de-fra-1-x.blook.network", true},                                // 1-char middle ok
		{"de-fra-1-.blook.network", false},                                // empty middle rejected
		{"de-fra-2-abc.blook.network", false},                             // wrong prefix
		{"de-fra-1-abc.evil.com", false},                                  // wrong suffix
		{"de-fra-1-" + strings.Repeat("a", 64) + ".blook.network", true},  // exactly 64 = max
		{"de-fra-1-" + strings.Repeat("a", 65) + ".blook.network", false}, // 65 > 64
		{"nope.com", false},                                               // no match
		{"", false},                                                       // empty
	}
	for _, tc := range cases {
		if got := c.matchServerName(tc.name); got != tc.want {
			t.Errorf("matchServerName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
