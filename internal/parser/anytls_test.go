package parser

import "testing"

func TestAnyTLSParser(t *testing.T) {
	node, err := ParseURL("anytls://secret@example.com:443/?insecure=1&sni=edge.example.com#Japan")
	if err != nil {
		t.Fatal(err)
	}
	if node.Type != "anytls" || node.Tag != "Japan" || node.Server != "example.com" || node.ServerPort != 443 {
		t.Fatalf("unexpected node: %#v", node)
	}
	if node.Extra["password"] != "secret" {
		t.Fatalf("password = %#v", node.Extra["password"])
	}
	tls, ok := node.Extra["tls"].(map[string]interface{})
	if !ok || tls["server_name"] != "edge.example.com" || tls["insecure"] != true {
		t.Fatalf("tls = %#v", node.Extra["tls"])
	}
}

func TestParseSubscriptionContentRejectsUnsupportedOnly(t *testing.T) {
	if _, err := ParseSubscriptionContent("unsupported://value"); err == nil {
		t.Fatal("expected error for subscription without supported nodes")
	}
}
