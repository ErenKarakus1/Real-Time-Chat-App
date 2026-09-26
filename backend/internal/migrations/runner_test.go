package migrations

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationFilesReturnsSQLFilesInOrder(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		"003_third.sql",
		"001_first.sql",
		"002_second.sql",
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("SELECT 1;"), 0o600); err != nil {
			t.Fatalf("write migration: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0o600); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}

	got, err := migrationFiles(dir)
	if err != nil {
		t.Fatalf("migration files: %v", err)
	}

	want := []string{
		filepath.Join(dir, "001_first.sql"),
		filepath.Join(dir, "002_second.sql"),
		filepath.Join(dir, "003_third.sql"),
	}
	if len(got) != len(want) {
		t.Fatalf("got %d files, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("file[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
