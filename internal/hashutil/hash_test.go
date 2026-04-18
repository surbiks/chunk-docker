package hashutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileSHA256(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := FileSHA256(path)
	if err != nil {
		t.Fatalf("FileSHA256 returned error: %v", err)
	}

	const want = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if got != want {
		t.Fatalf("FileSHA256 = %s, want %s", got, want)
	}
}

func TestReaderSHA256(t *testing.T) {
	t.Parallel()

	got, err := ReaderSHA256(strings.NewReader("abc"))
	if err != nil {
		t.Fatalf("ReaderSHA256 returned error: %v", err)
	}

	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Fatalf("ReaderSHA256 = %s, want %s", got, want)
	}
}
