package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Counters mirrors the cumulative counters exposed by sing-box's Clash API.
type Counters struct {
	UploadTotal   uint64 `json:"uploadTotal"`
	DownloadTotal uint64 `json:"downloadTotal"`
}

type Snapshot struct {
	Upload          uint64    `json:"upload"`
	Download        uint64    `json:"download"`
	SessionUpload   uint64    `json:"session_upload"`
	SessionDownload uint64    `json:"session_download"`
	UpdatedAt       time.Time `json:"updated_at"`
	Available       bool      `json:"available"`
}

type state struct {
	Upload       uint64    `json:"upload"`
	Download     uint64    `json:"download"`
	LastUpload   uint64    `json:"last_upload"`
	LastDownload uint64    `json:"last_download"`
	LastPID      int       `json:"last_pid"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Tracker struct {
	mu        sync.RWMutex
	path      string
	state     state
	port      func() int
	secret    func() string
	pid       func() int
	client    *http.Client
	cancel    context.CancelFunc
	done      chan struct{}
	dirty     bool
	lastSaved time.Time
}

func New(dataDir string, port func() int, secret func() string, pid func() int) (*Tracker, error) {
	t := &Tracker{path: filepath.Join(dataDir, "usage.json"), port: port, secret: secret, pid: pid, client: &http.Client{Timeout: 3 * time.Second}}
	data, err := os.ReadFile(t.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &t.state); err != nil {
			return nil, fmt.Errorf("解析流量历史失败: %w", err)
		}
		t.lastSaved = time.Now()
	}
	return t, nil
}

func (t *Tracker) Start() {
	if t == nil {
		return
	}
	t.mu.Lock()
	if t.cancel != nil {
		t.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel, t.done = cancel, make(chan struct{})
	done := t.done
	t.mu.Unlock()
	go func() {
		defer close(done)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			_ = t.Poll(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (t *Tracker) Stop() {
	if t == nil {
		return
	}
	t.mu.Lock()
	cancel, done := t.cancel, t.done
	t.cancel = nil
	t.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
	_ = t.flush()
}

func (t *Tracker) Poll(ctx context.Context) error {
	pid := t.pid()
	if pid <= 0 {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/connections", t.port()), nil)
	if err != nil {
		return err
	}
	if secret := t.secret(); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	response, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Clash API 返回 HTTP %d", response.StatusCode)
	}
	var counters Counters
	if err := json.NewDecoder(response.Body).Decode(&counters); err != nil {
		return err
	}
	return t.Observe(pid, counters, time.Now())
}

// Observe folds one process-local sample into durable totals.
func (t *Tracker) Observe(pid int, counters Counters, at time.Time) error {
	if pid <= 0 {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	next := t.state
	if pid != next.LastPID || counters.UploadTotal < next.LastUpload || counters.DownloadTotal < next.LastDownload {
		next.Upload += counters.UploadTotal
		next.Download += counters.DownloadTotal
	} else {
		next.Upload += counters.UploadTotal - next.LastUpload
		next.Download += counters.DownloadTotal - next.LastDownload
	}
	next.LastUpload, next.LastDownload, next.LastPID, next.UpdatedAt = counters.UploadTotal, counters.DownloadTotal, pid, at
	t.state = next
	t.dirty = true
	if t.lastSaved.IsZero() || at.Sub(t.lastSaved) >= time.Minute {
		if err := t.save(t.state); err != nil {
			return err
		}
		t.dirty = false
		t.lastSaved = at
	}
	return nil
}

func (t *Tracker) flush() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.dirty {
		return nil
	}
	if err := t.save(t.state); err != nil {
		return err
	}
	t.dirty = false
	t.lastSaved = time.Now()
	return nil
}

func (t *Tracker) Snapshot() Snapshot {
	if t == nil {
		return Snapshot{}
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return Snapshot{Upload: t.state.Upload, Download: t.state.Download, SessionUpload: t.state.LastUpload, SessionDownload: t.state.LastDownload, UpdatedAt: t.state.UpdatedAt, Available: !t.state.UpdatedAt.IsZero()}
}

func (t *Tracker) save(value state) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(t.path), ".usage-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), t.path)
}
