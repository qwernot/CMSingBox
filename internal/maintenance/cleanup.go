package maintenance

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Preview struct {
	LogsBytes      int64 `json:"logs_bytes"`
	TemporaryBytes int64 `json:"temporary_bytes"`
	Files          int   `json:"files"`
}
type Cleaner struct{ dataDir string }

func New(dataDir string) *Cleaner { return &Cleaner{dataDir: filepath.Clean(dataDir)} }

func (c *Cleaner) candidates() (logs []string, temporary []string) {
	logs = []string{filepath.Join(c.dataDir, "logs", "sbm.log"), filepath.Join(c.dataDir, "logs", "singbox.log")}
	for _, dir := range []string{"tmp", "cache", "downloads"} {
		root := filepath.Join(c.dataDir, dir)
		entries, _ := os.ReadDir(root)
		for _, entry := range entries {
			temporary = append(temporary, filepath.Join(root, entry.Name()))
		}
	}
	entries, _ := os.ReadDir(filepath.Join(c.dataDir, "generated"))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") || strings.HasSuffix(entry.Name(), ".bak") {
			temporary = append(temporary, filepath.Join(c.dataDir, "generated", entry.Name()))
		}
	}
	return
}

func sizeOf(path string) (int64, int) {
	var size int64
	var files int
	_ = filepath.WalkDir(path, func(_ string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			if info, e := entry.Info(); e == nil {
				size += info.Size()
				files++
			}
		}
		return nil
	})
	return size, files
}
func (c *Cleaner) Preview() Preview {
	logs, temp := c.candidates()
	var result Preview
	for _, path := range logs {
		size, count := sizeOf(path)
		result.LogsBytes += size
		result.Files += count
	}
	for _, path := range temp {
		size, count := sizeOf(path)
		result.TemporaryBytes += size
		result.Files += count
	}
	return result
}
func (c *Cleaner) Clean(logsEnabled, temporaryEnabled bool) error {
	logs, temp := c.candidates()
	if logsEnabled {
		for _, path := range logs {
			if _, err := os.Stat(path); err == nil {
				if err := os.Truncate(path, 0); err != nil {
					return err
				}
			}
		}
	}
	if temporaryEnabled {
		for _, path := range temp {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
		}
	}
	return nil
}
