package zerotrue

import "strings"

const defaultAPIBaseURL = "https://api.zerotrue.app"

// testAPIBaseURL is set by unit tests only (see setTestAPIBaseURL in client_test.go).
var testAPIBaseURL string

func effectiveAPIBaseURL() string {
	if testAPIBaseURL != "" {
		return strings.TrimRight(testAPIBaseURL, "/")
	}
	return defaultAPIBaseURL
}
