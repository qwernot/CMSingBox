package storage

import (
	"errors"
	"testing"
)

func TestSubscriptionLimitCountsLinksNotNodes(t *testing.T) {
	store, err := NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manyNodes := make([]Node, 100)
	if err := store.AddSubscription(Subscription{ID: "one", Nodes: manyNodes}); err != nil {
		t.Fatalf("nodes inside one subscription must not consume quota: %v", err)
	}
	if err := store.AddManualNode(ManualNode{ID: "manual", Enabled: true}); err != nil {
		t.Fatalf("manual node must not consume subscription quota: %v", err)
	}
	if err := store.AddSubscription(Subscription{ID: "two"}); !errors.Is(err, ErrSubscriptionLimitExceeded) {
		t.Fatalf("expected subscription limit error, got %v", err)
	}
	store.SetSubscriptionLimit(2)
	if err := store.AddSubscription(Subscription{ID: "two"}); err != nil {
		t.Fatalf("licensed quota should allow second link: %v", err)
	}
}

func TestGetSubscriptionReturnsCopy(t *testing.T) {
	store, _ := NewJSONStore(t.TempDir())
	_ = store.AddSubscription(Subscription{ID: "one", Name: "original", Nodes: []Node{{Tag: "node"}}})
	copy := store.GetSubscription("one")
	copy.Name = "changed"
	copy.Nodes[0].Tag = "changed"
	stored := store.GetSubscription("one")
	if stored.Name != "original" || stored.Nodes[0].Tag != "node" {
		t.Fatalf("caller mutated stored subscription: %#v", stored)
	}
}
