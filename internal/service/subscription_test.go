package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cmsingbox.local/cmsingbox/internal/storage"
)

func TestRefreshAllReportsFailedSubscription(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer upstream.Close()

	store, err := storage.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.SetSubscriptionLimit(-1)
	if err := store.AddSubscription(storage.Subscription{ID: "broken", Name: "测试订阅", URL: upstream.URL, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	err = NewSubscriptionService(store).RefreshAll()
	if err == nil || !strings.Contains(err.Error(), "测试订阅") {
		t.Fatalf("RefreshAll() error = %v, want named subscription failure", err)
	}
}

func TestSchedulerAppliesSuccessfulPartialRefreshes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer upstream.Close()

	store, err := storage.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.SetSubscriptionLimit(-1)
	if err := store.AddSubscription(storage.Subscription{ID: "broken", Name: "测试订阅", URL: upstream.URL, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	called := false
	scheduler := NewScheduler(store, NewSubscriptionService(store))
	scheduler.SetUpdateCallback(func() error {
		called = true
		return nil
	})
	scheduler.updateSubscriptions()
	if !called {
		t.Fatal("scheduler skipped config apply after a partial refresh failure")
	}
}

func TestFilterInformationalNodes(t *testing.T) {
	nodes := []storage.Node{
		{Tag: "剩余流量：96 GB"},
		{Tag: "套餐到期：长期有效"},
		{Tag: "过滤掉15条线路"},
		{Tag: "日本JP-A"},
	}
	got := filterInformationalNodes(nodes)
	if len(got) != 1 || got[0].Tag != "日本JP-A" {
		t.Fatalf("filtered nodes = %#v", got)
	}
}
