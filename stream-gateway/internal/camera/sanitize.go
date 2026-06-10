package camera

import "net/url"

// sanitizeRTSPURL replaces the password portion of an RTSP URL with "***".
// Returns the original URL unchanged if parsing fails or there is no password.
//
//	rtsp://admin:secret@host:554/path → rtsp://admin:***@host:554/path
func sanitizeRTSPURL(rawURL string) string {
	if rawURL == "" {
		return rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if u.User == nil {
		return rawURL
	}
	if _, hasPW := u.User.Password(); !hasPW {
		return rawURL
	}

	// Rebuild userinfo with masked password, preserving everything else.
	u.User = url.UserPassword(u.User.Username(), "***")
	return u.String()
}
