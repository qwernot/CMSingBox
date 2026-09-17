package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReferenceRulePresetOrder(t *testing.T) {
	groups := DefaultRuleGroups()
	want := []string{"cn-cdn", "youtube", "telegram", "spotify", "netflix", "ai-services", "github", "google", "steam", "cn", "non-cn"}
	if len(groups) != len(want) {
		t.Fatalf("got %d groups, want %d", len(groups), len(want))
	}
	for i, id := range want {
		if groups[i].ID != id || !groups[i].Enabled {
			t.Fatalf("group %d = %+v, want enabled %s", i, groups[i], id)
		}
	}
}

func TestLegacyRuleGroupsUpgradeWithoutRemovingUserRules(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	store.data.RulePresetVersion = 0
	store.data.RuleGroups = []RuleGroup{{ID: "ad-block", Name: "广告拦截", Enabled: true}}
	store.data.Rules = []Rule{{ID: "custom", Name: "自定义", Enabled: true}}
	data, err := json.Marshal(store.data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewJSONStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.data.RulePresetVersion != CurrentRulePresetVersion || len(reloaded.GetRuleGroups()) != 11 {
		t.Fatalf("preset not upgraded: %+v", reloaded.data)
	}
	if len(reloaded.data.Rules) != 1 || reloaded.data.Rules[0].ID != "custom" {
		t.Fatal("custom rule was removed during migration")
	}
}
