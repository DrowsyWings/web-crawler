package normalize

import (
	"net/url"
	"strings"
)

// Normalize returns a canonical form of rawURL for deduplication, or rawURL
// unchanged if it cannot be parsed.
func Normalize(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	if (u.Scheme == "http" && strings.HasSuffix(u.Host, ":80")) ||
		(u.Scheme == "https" && strings.HasSuffix(u.Host, ":443")) {
		u.Host = u.Hostname()
	}

	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")

	return u.String()
}
