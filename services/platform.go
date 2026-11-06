package services

import (
	"os"
	"regexp"
	"strings"
)

var platformIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,23}$`)

// PlatformID is configured by the deployment, never by a request or a front.
func PlatformID() string {
	id := strings.TrimSpace(os.Getenv("PLATFORM_ID"))
	if id == "" {
		return ""
	}
	if !platformIDPattern.MatchString(id) {
		panic("Invalid PLATFORM_ID")
	}
	return id
}
