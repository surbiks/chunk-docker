package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", path, err)
	}
	return nil
}

func CreateRunDir(baseDir, prefix string) (string, error) {
	if err := EnsureDir(baseDir); err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	fullPath := filepath.Join(baseDir, name)
	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		return "", fmt.Errorf("create run directory %q: %w", fullPath, err)
	}

	return fullPath, nil
}

func LinkOrCopy(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source %q: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination %q: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy %q to %q: %w", src, dst, err)
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync %q: %w", dst, err)
	}

	return nil
}

func RemoveAll(path string) error {
	if path == "" {
		return nil
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove %q: %w", path, err)
	}
	return nil
}
