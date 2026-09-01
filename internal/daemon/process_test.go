package daemon

import "testing"

func TestMatchesManagedCommandRequiresOwnConfig(t *testing.T) {
	pm := &ProcessManager{configPath: "/var/lib/cmsingbox/generated/config.json"}
	if !pm.matchesManagedCommand([]string{"/var/lib/cmsingbox/bin/sing-box", "run", "-c", "/var/lib/cmsingbox/generated/config.json"}) {
		t.Fatal("own sing-box process was not recognized")
	}
	if pm.matchesManagedCommand([]string{"/data/bin/sing-box", "run", "-c", "/data/generated/config.json"}) {
		t.Fatal("Docker sing-box process was incorrectly recognized by native instance")
	}
	if pm.matchesManagedCommand([]string{"/var/lib/cmsingbox/bin/sing-box", "version"}) {
		t.Fatal("short-lived version process was incorrectly recognized")
	}
}
