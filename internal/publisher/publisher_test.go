package publisher

import (
	"reflect"
	"testing"

	"github.com/saeid/chunkdocker/internal/chunker"
)

func TestRenderDockerfileUsesConfiguredBaseImage(t *testing.T) {
	t.Parallel()

	lines := renderDockerfile("alpine:3.20", []chunker.ChunkArtifact{
		{FileName: "chunk-00000.bin", PathInImage: "/chunks/chunk-00000.bin"},
		{FileName: "chunk-00001.bin", PathInImage: "/chunks/chunk-00001.bin"},
	})

	want := []string{
		"FROM alpine:3.20",
		"COPY chunk-00000.bin /chunks/chunk-00000.bin",
		"COPY chunk-00001.bin /chunks/chunk-00001.bin",
	}

	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("renderDockerfile() = %v, want %v", lines, want)
	}
}
