package cli

import "testing"

func TestParseGoBin(t *testing.T) {
	for name, tc := range map[string]struct {
		out  string
		want string
	}{
		"gobin set":   {"/home/u/bin\n/home/u/go\n", "/home/u/bin/usagely"},
		"gopath only": {"\n/home/u/go\n", "/home/u/go/bin/usagely"},
		"neither set": {"\n\n", "usagely"},
		"empty":       {"", "usagely"},
	} {
		if got := parseGoBin(tc.out); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}
