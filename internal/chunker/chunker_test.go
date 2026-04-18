package chunker

import "testing"

func TestGroupChunksSingleChunkMode(t *testing.T) {
	t.Parallel()

	chunks := []ChunkArtifact{
		{Index: 0, FileName: "chunk-00000.bin"},
		{Index: 1, FileName: "chunk-00001.bin"},
	}

	groups := GroupChunks(chunks, 1, "docker.io", "example/chunks", "v1")
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Tag != "v1-00000" || groups[1].Tag != "v1-00001" {
		t.Fatalf("unexpected single chunk tags: %+v", groups)
	}
}

func TestGroupChunksGroupedMode(t *testing.T) {
	t.Parallel()

	chunks := []ChunkArtifact{
		{Index: 0}, {Index: 1}, {Index: 2}, {Index: 3}, {Index: 4}, {Index: 5},
	}

	groups := GroupChunks(chunks, 5, "docker.io", "example/chunks", "v1")
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Tag != "v1-g00000" || groups[1].Tag != "v1-g00001" {
		t.Fatalf("unexpected grouped tags: %+v", groups)
	}
	if len(groups[0].Chunks) != 5 || len(groups[1].Chunks) != 1 {
		t.Fatalf("unexpected group sizes: %+v", groups)
	}
}
