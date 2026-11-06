package controllers

import "testing"

func TestOnlifePresenceOrigin(t *testing.T) {
	t.Setenv("PRESENCE_ALLOWED_ORIGINS", "")
	for _, origin := range []string{"https://educacaoonlife.gpedu.com.br", "https://web.uniusp.gpedu.com.br"} {
		if !presenceOriginAllowed(origin) {
			t.Fatalf("origin rejected: %s", origin)
		}
	}
	for _, origin := range []string{"https://educacaoonlife.gpedu.com.br.evil.test", "http://educacaoonlife.gpedu.com.br", "https://educacaoonlife.gpedu.com.br/path"} {
		if presenceOriginAllowed(origin) {
			t.Fatalf("unsafe origin accepted: %s", origin)
		}
	}
	t.Setenv("PRESENCE_ALLOWED_ORIGINS", "https://custom.example")
	if presenceOriginAllowed("https://educacaoonlife.gpedu.com.br") {
		t.Fatal("explicit allowlist ignored")
	}
}
