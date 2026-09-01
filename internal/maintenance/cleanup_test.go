package maintenance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewAndClean(t *testing.T) {
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "logs"), 0700)
	_ = os.MkdirAll(filepath.Join(root, "tmp"), 0700)
	_ = os.WriteFile(filepath.Join(root, "logs", "sbm.log"), []byte("log"), 0600)
	_ = os.WriteFile(filepath.Join(root, "tmp", "item"), []byte("temp"), 0600)
	cleaner := New(root)
	preview := cleaner.Preview()
	if preview.LogsBytes != 3 || preview.TemporaryBytes != 4 {
		t.Fatalf("unexpected preview %#v", preview)
	}
	if err := cleaner.Clean(true, true); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(filepath.Join(root, "logs", "sbm.log")); info.Size() != 0 {
		t.Fatal("log was not truncated")
	}
	if _, err := os.Stat(filepath.Join(root, "tmp", "item")); !os.IsNotExist(err) {
		t.Fatal("temporary file was not deleted")
	}
}
