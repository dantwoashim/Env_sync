package store

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T, maxVersions int) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	return &Store{
		maxVersions: maxVersions,
		baseDir:     filepath.Join(tmpDir, "store"),
	}
}

func testEncryptionKey() [32]byte {
	var key [32]byte
	copy(key[:], []byte("test-key-32-bytes-exactly-padded!"))
	return key
}

func TestSaveAndRestore(t *testing.T) {
	s := newTestStore(t, 10)
	key := testEncryptionKey()
	project := "test-project"

	original := []byte("SECRET_KEY=abc123\nDB_HOST=localhost")

	if err := s.Save(project, original, 1, key); err != nil {
		t.Fatalf("save: %v", err)
	}

	restored, err := s.Restore(project, 1, key)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}

	if string(restored) != string(original) {
		t.Errorf("restored = %q, want %q", restored, original)
	}
}

func TestListVersions(t *testing.T) {
	s := newTestStore(t, 10)
	key := testEncryptionKey()
	project := "test-project"

	for i := 1; i <= 5; i++ {
		if err := s.Save(project, []byte("data"), i, key); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	versions, err := s.List(project)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(versions) != 5 {
		t.Errorf("got %d versions, want 5", len(versions))
	}

	// Newest first
	if versions[0].Sequence != 5 {
		t.Errorf("first version seq = %d, want 5", versions[0].Sequence)
	}
}

func TestLatestVersion(t *testing.T) {
	s := newTestStore(t, 10)
	key := testEncryptionKey()
	project := "test-project"

	s.Save(project, []byte("v1"), 1, key)
	s.Save(project, []byte("v2"), 2, key)

	latest, err := s.Latest(project)
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest == nil {
		t.Fatal("latest is nil")
	}
	if latest.Sequence != 2 {
		t.Errorf("latest seq = %d, want 2", latest.Sequence)
	}
}

func TestRotation(t *testing.T) {
	s := newTestStore(t, 3) // Keep only 3 versions
	key := testEncryptionKey()
	project := "test-project"

	for i := 1; i <= 6; i++ {
		if err := s.Save(project, []byte("data"), i, key); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}

	versions, err := s.List(project)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(versions) != 3 {
		t.Errorf("got %d versions after rotation, want 3", len(versions))
	}

	// Should keep sequences 4, 5, 6
	if versions[0].Sequence != 6 {
		t.Errorf("newest = %d, want 6", versions[0].Sequence)
	}
	if versions[2].Sequence != 4 {
		t.Errorf("oldest = %d, want 4", versions[2].Sequence)
	}
}

func TestRestoreWrongKey(t *testing.T) {
	s := newTestStore(t, 10)
	key := testEncryptionKey()
	project := "test-project"

	s.Save(project, []byte("secret data"), 1, key)

	var wrongKey [32]byte
	copy(wrongKey[:], []byte("wrong-key-32-bytes-exactly-pad!!"))

	_, err := s.Restore(project, 1, wrongKey)
	if err == nil {
		t.Error("expected error with wrong key")
	}
}

func TestRestoreNonexistentVersion(t *testing.T) {
	s := newTestStore(t, 10)
	key := testEncryptionKey()

	_, err := s.Restore("nonexistent", 1, key)
	if err == nil {
		t.Error("expected error for nonexistent version")
	}
}

func TestLatestEmpty(t *testing.T) {
	s := newTestStore(t, 10)

	latest, err := s.Latest("empty-project")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest != nil {
		t.Error("expected nil for empty project")
	}
}

func TestEncryptedFileFormat(t *testing.T) {
	s := newTestStore(t, 10)
	key := testEncryptionKey()
	project := "test-project"

	s.Save(project, []byte("test"), 1, key)

	// Check directory structure
	dir := s.projectDir(project)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 file, got %d", len(entries))
	}
	if filepath.Ext(entries[0].Name()) != ".enc" {
		t.Errorf("expected .enc extension, got %s", entries[0].Name())
	}
}
