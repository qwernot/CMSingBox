package api

import (
	"os"
	"path/filepath"
	"testing"

	"cmsingbox.local/cmsingbox/internal/storage"
)

func TestBackHomeCertificateIsGeneratedAndReused(t *testing.T) {
	store, err := storage.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{store: store}
	settings := storage.DefaultSettings()
	settings.BackHomeEnabled = true
	settings.BackHomeServer = "home.example.com"
	settings.BackHomePassword = "secret"
	if err := server.ensureBackHomeCertificate(settings); err != nil {
		t.Fatal(err)
	}
	if settings.BackHomeCertPath == "" || settings.BackHomeKeyPath == "" {
		t.Fatal("missing certificate paths")
	}
	first, err := os.ReadFile(settings.BackHomeCertPath)
	if err != nil {
		t.Fatal(err)
	}
	if mode := fileMode(t, settings.BackHomeKeyPath); mode != 0600 {
		t.Fatalf("key mode = %o", mode)
	}
	settings.BackHomeCertPath, settings.BackHomeKeyPath = "", ""
	if err := server.ensureBackHomeCertificate(settings); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(settings.BackHomeCertPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("certificate unexpectedly changed")
	}
	if filepath.Dir(settings.BackHomeKeyPath) != filepath.Dir(settings.BackHomeCertPath) {
		t.Fatal("certificate files separated")
	}
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}
