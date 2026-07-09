package normalize

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"trailing slash on root", "https://example.com/", "https://example.com"},
		{"no trailing slash", "https://example.com", "https://example.com"},
		{"trailing slash on path", "https://example.com/foo/", "https://example.com/foo"},
		{"uppercase scheme and host", "HTTPS://Example.COM/Path", "https://example.com/Path"},
		{"path case preserved", "https://example.com/PaGe", "https://example.com/PaGe"},
		{"default http port stripped", "http://example.com:80/", "http://example.com"},
		{"default https port stripped", "https://example.com:443/foo", "https://example.com/foo"},
		{"non-default port kept", "http://example.com:8080/", "http://example.com:8080"},
		{"fragment dropped", "https://example.com/page#section", "https://example.com/page"},
		{"query preserved", "https://example.com/search?q=go", "https://example.com/search?q=go"},
		{"unparseable returned unchanged", "http://example.com/%zz", "http://example.com/%zz"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Normalize(tc.in); got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
