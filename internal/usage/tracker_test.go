package usage

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestTotalsSurviveCoreAndManagerRestarts(t *testing.T) {
	dir := t.TempDir()
	newTracker := func() *Tracker {
		tracker, err := New(dir, func() int { return 9090 }, func() string { return "" }, func() int { return 1 })
		if err != nil {
			t.Fatal(err)
		}
		return tracker
	}
	tracker := newTracker()
	for _, sample := range []struct {
		pid      int
		up, down uint64
	}{{10, 100, 200}, {10, 150, 270}, {11, 20, 30}} {
		if err := tracker.Observe(sample.pid, Counters{UploadTotal: sample.up, DownloadTotal: sample.down}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	tracker.Stop()
	tracker = newTracker()
	if err := tracker.Observe(11, Counters{UploadTotal: 25, DownloadTotal: 35}, time.Now()); err != nil {
		t.Fatal(err)
	}
	snapshot := tracker.Snapshot()
	if snapshot.Upload != 175 || snapshot.Download != 305 || snapshot.SessionUpload != 25 {
		t.Fatalf("unexpected durable totals: %+v", snapshot)
	}
}

func TestPollReadsAuthenticatedClashCounters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connections" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("unexpected request: %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"uploadTotal":42,"downloadTotal":84}`))
	}))
	defer server.Close()
	_, portText, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portText)
	tracker, err := New(t.TempDir(), func() int { return port }, func() string { return "secret" }, func() int { return 42 })
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.Poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if snapshot := tracker.Snapshot(); snapshot.Upload != 42 || snapshot.Download != 84 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}
