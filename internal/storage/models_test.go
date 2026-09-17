package storage

import "testing"

func TestGatewayDefaults(t *testing.T) {
	settings := DefaultSettings()
	if !settings.DNSEnabled || settings.DNSListen != "0.0.0.0:53" {
		t.Fatalf("unexpected DNS defaults: %+v", settings)
	}
	if settings.ClashAPISecret != "" {
		t.Fatal("default Clash API must be passwordless")
	}
}
