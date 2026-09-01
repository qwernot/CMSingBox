package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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

func TestRestartIgnoresPreviousProcessExit(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "sing-box")
	script := `#!/bin/sh
case "$1" in
  check) exit 0 ;;
  version) echo "sing-box version 1.13.0"; exit 0 ;;
  run) trap 'exit 0' TERM; while :; do sleep 1; done ;;
esac
`
	if err := os.WriteFile(fake, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "config.json")
	if err := os.WriteFile(config, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	pm := &ProcessManager{
		singboxPath: fake,
		configPath:  config,
		dataDir:     dir,
		pidFile:     filepath.Join(dir, "singbox.pid"),
		desiredFile: filepath.Join(dir, "singbox.enabled"),
		maxLogs:     10,
	}
	if err := pm.Start(); err != nil {
		t.Fatal(err)
	}
	firstPID := pm.GetPID()
	if err := pm.Restart(); err != nil {
		t.Fatal(err)
	}
	secondPID := pm.GetPID()
	if firstPID == secondPID || secondPID <= 0 {
		t.Fatalf("restart PIDs = %d -> %d", firstPID, secondPID)
	}
	time.Sleep(150 * time.Millisecond)
	pm.mu.RLock()
	running, currentPID := pm.running, pm.pid
	pm.mu.RUnlock()
	if !running || currentPID != secondPID {
		t.Fatalf("old exit cleared new state: running=%v pid=%d, want pid=%d", running, currentPID, secondPID)
	}
	if err := pm.Stop(); err != nil {
		t.Fatal(err)
	}
}
