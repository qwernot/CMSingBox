package firewall

import (
	"strings"
	"testing"
)

func TestBuildNFTables(t *testing.T) {
	script, err := BuildNFTables(Config{Port: 7893, BypassCIDRs: []string{"10.0.0.0/8", "192.168.0.0/16"}})
	if err != nil || !strings.Contains(script, "tproxy to :7893") || !strings.Contains(script, "10.0.0.0/8") {
		t.Fatalf("unexpected script: %v %s", err, script)
	}
	if _, err := BuildNFTables(Config{Port: 0}); err == nil {
		t.Fatal("invalid port must fail")
	}
	if _, err := BuildNFTables(Config{Port: 7893, BypassCIDRs: []string{"invalid"}}); err == nil {
		t.Fatal("invalid CIDR must fail")
	}
}
