package storage

import "testing"

func TestRuleSetBaseURLMigratesFromThirdPartyMirror(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	settings := store.GetSettings()
	settings.RuleSetBaseURL = "https://github.com/lyc8503/sing-box-rules/raw/rule-set-geosite"
	if err := store.UpdateSettings(settings); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewJSONStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.GetSettings().RuleSetBaseURL; got != DefaultRuleSetBaseURL {
		t.Fatalf("ruleset URL = %q, want %q", got, DefaultRuleSetBaseURL)
	}
}
