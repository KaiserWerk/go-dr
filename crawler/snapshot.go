package crawler

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SnapshotStore persists fetched payloads for reproducible parsing and tests.
type SnapshotStore interface {
	Save(sourceName, documentID string, payload []byte) (string, error)
}

// FileSnapshotStore stores snapshots on disk.
type FileSnapshotStore struct {
	BaseDir string
}

var nonFileChar = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// Save writes payload into <base>/<source>/<timestamp>_<id>.snapshot.
func (s FileSnapshotStore) Save(sourceName, documentID string, payload []byte) (string, error) {
	base := s.BaseDir
	if strings.TrimSpace(base) == "" {
		base = "snapshots"
	}

	dir := filepath.Join(base, cleanPathPart(sourceName))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	fileName := fmt.Sprintf("%s_%s.snapshot",
		time.Now().UTC().Format("20060102T150405Z"),
		cleanPathPart(documentID),
	)
	if strings.TrimSpace(documentID) == "" {
		fileName = fmt.Sprintf("%s.snapshot", time.Now().UTC().Format("20060102T150405Z"))
	}

	fullPath := filepath.Join(dir, fileName)
	if err := os.WriteFile(fullPath, payload, 0o644); err != nil {
		return "", err
	}

	return fullPath, nil
}

func cleanPathPart(v string) string {
	trimmed := strings.TrimSpace(v)
	if trimmed == "" {
		return "unknown"
	}
	cleaned := nonFileChar.ReplaceAllString(trimmed, "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
}
