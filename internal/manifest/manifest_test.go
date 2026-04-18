package manifest

import (
	"path/filepath"
	"testing"
	"time"
)

func TestManifestRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	input := &Manifest{
		Version:           CurrentVersion,
		CreatedAt:         time.Now().UTC().Truncate(time.Second),
		FileName:          "ubuntu.iso",
		FileSize:          123,
		FileSHA256:        "abc",
		ChunkSize:         50,
		ChunkCount:        2,
		ChunksPerImage:    2,
		Registry:          "docker.io",
		Repository:        "example/chunks",
		Release:           "v1",
		ImageChunkBaseDir: "/chunks",
		Groups: []GroupEntry{
			{GroupIndex: 0, Tag: "v1-g00000", Image: "docker.io/example/chunks:v1-g00000", ChunkIndexes: []int{0, 1}},
		},
		Chunks: []ChunkEntry{
			{Index: 0, GroupIndex: 0, GroupTag: "v1-g00000", Image: "docker.io/example/chunks:v1-g00000", PathInImage: "/chunks/chunk-00000.bin", Size: 50, SHA256: "111"},
			{Index: 1, GroupIndex: 0, GroupTag: "v1-g00000", Image: "docker.io/example/chunks:v1-g00000", PathInImage: "/chunks/chunk-00001.bin", Size: 73, SHA256: "222"},
		},
	}

	if err := Save(path, input); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	output, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if output.FileName != input.FileName || output.ChunkCount != input.ChunkCount {
		t.Fatalf("round trip mismatch: got %+v want %+v", output, input)
	}
}
