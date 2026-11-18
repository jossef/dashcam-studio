package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyFolderIsMoved(t *testing.T) {
	base := t.TempDir()
	old := filepath.Join(base, legacyDir)
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "favorites.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := appFolder(base)
	if got, err := os.ReadFile(filepath.Join(dir, "favorites.json")); err != nil || string(got) != `{"version":1}` {
		t.Fatalf("favorites not carried over: %q %v", got, err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old folder still there: %v", err)
	}
}

func TestExistingFolderWins(t *testing.T) {
	base := t.TempDir()
	for _, d := range []string{legacyDir, appDir} {
		if err := os.MkdirAll(filepath.Join(base, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	appFolder(base)
	if _, err := os.Stat(filepath.Join(base, legacyDir)); err != nil {
		t.Fatal("old folder should be left alone when the new one exists")
	}
}
