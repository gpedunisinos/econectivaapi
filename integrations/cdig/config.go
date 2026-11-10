// Package cdig implements optional, server-side CDIG content synchronization.
package cdig

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Enabled                                    bool
	BaseURL, SourceKey, Token, Email, Password string
	Interval                                   time.Duration
	ActorID                                    uint
}

// Load never returns secrets in diagnostics. Missing/invalid settings disable the
// integration only; they must never prevent the platform from starting.
func Load() (Config, string) { return FromEnv(os.Getenv) }
func FromEnv(get func(string) string) (Config, string) {
	c := Config{}
	if get("CDIG_SYNC_ENABLED") != "true" {
		return c, "disabled"
	}
	c.BaseURL = strings.TrimRight(strings.TrimSpace(get("CDIG_BASE_URL")), "/")
	c.SourceKey = strings.TrimSpace(get("CDIG_SOURCE_KEY"))
	c.Token = strings.TrimSpace(get("CDIG_API_TOKEN"))
	c.Email = strings.TrimSpace(get("CDIG_EMAIL"))
	c.Password = get("CDIG_PASSWORD")
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return Config{}, "CDIG_BASE_URL must be an HTTPS origin"
	}
	if c.SourceKey == "" || len(c.SourceKey) > 100 {
		return Config{}, "CDIG_SOURCE_KEY is required (up to 100 characters)"
	}
	if c.Token == "" && (c.Email == "" || c.Password == "") {
		return Config{}, "configure CDIG_API_TOKEN or both CDIG_EMAIL and CDIG_PASSWORD"
	}
	actor, err := strconv.ParseUint(get("CDIG_IMPORT_USER_ID"), 10, 32)
	if err != nil || actor == 0 {
		return Config{}, "CDIG_IMPORT_USER_ID is required"
	}
	c.ActorID = uint(actor)
	if raw := get("CDIG_SYNC_INTERVAL_HOURS"); raw != "" && raw != "0" {
		hours, e := strconv.Atoi(raw)
		if e != nil || hours < 1 || hours > 168 {
			return Config{}, "CDIG_SYNC_INTERVAL_HOURS must be 0 or between 1 and 168"
		}
		c.Interval = time.Duration(hours) * time.Hour
	}
	c.Enabled = true
	return c, "enabled"
}
func (c Config) SourceID() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(c.BaseURL+"\n"+c.SourceKey)))
}
