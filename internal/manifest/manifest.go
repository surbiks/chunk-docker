package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const CurrentVersion = 1

type Manifest struct {
	Version           int          `json:"version"`
	CreatedAt         time.Time    `json:"created_at"`
	FileName          string       `json:"file_name"`
	FileSize          int64        `json:"file_size"`
	FileSHA256        string       `json:"file_sha256"`
	ChunkSize         int64        `json:"chunk_size"`
	ChunkCount        int          `json:"chunk_count"`
	ChunksPerImage    int          `json:"chunks_per_image"`
	PublishRegistry   string       `json:"publish_registry"`
	Registry          string       `json:"registry"`
	Repository        string       `json:"repository"`
	Release           string       `json:"release"`
	ImageChunkBaseDir string       `json:"image_chunk_base_dir"`
	Groups            []GroupEntry `json:"groups"`
	Chunks            []ChunkEntry `json:"chunks"`
}

type GroupEntry struct {
	GroupIndex   int    `json:"group_index"`
	Tag          string `json:"tag"`
	Image        string `json:"image"`
	ChunkIndexes []int  `json:"chunk_indexes"`
}

type ChunkEntry struct {
	Index       int    `json:"index"`
	GroupIndex  int    `json:"group_index"`
	GroupTag    string `json:"group_tag"`
	Image       string `json:"image"`
	PathInImage string `json:"path_in_image"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}

func (m *Manifest) Validate() error {
	if m.Version != CurrentVersion {
		return fmt.Errorf("unsupported manifest version %d", m.Version)
	}
	if m.FileName == "" {
		return fmt.Errorf("file_name is required")
	}
	if m.FileSize < 0 {
		return fmt.Errorf("file_size must be non-negative")
	}
	if m.FileSHA256 == "" {
		return fmt.Errorf("file_sha256 is required")
	}
	if m.ChunkSize <= 0 {
		return fmt.Errorf("chunk_size must be greater than zero")
	}
	if m.ChunkCount != len(m.Chunks) {
		return fmt.Errorf("chunk_count %d does not match chunk list length %d", m.ChunkCount, len(m.Chunks))
	}
	if m.ChunksPerImage <= 0 {
		return fmt.Errorf("chunks_per_image must be at least 1")
	}
	if m.PublishRegistry == "" {
		return fmt.Errorf("publish_registry is required")
	}
	if m.Registry == "" {
		return fmt.Errorf("registry is required")
	}
	if m.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if m.Release == "" {
		return fmt.Errorf("release is required")
	}
	if len(m.Chunks) == 0 {
		return fmt.Errorf("chunks list must not be empty")
	}
	if len(m.Groups) > 0 {
		for _, group := range m.Groups {
			if group.Image == "" || group.Tag == "" {
				return fmt.Errorf("group %d is missing image metadata", group.GroupIndex)
			}
			for _, chunkIndex := range group.ChunkIndexes {
				if chunkIndex < 0 || chunkIndex >= len(m.Chunks) {
					return fmt.Errorf("group %d references invalid chunk index %d", group.GroupIndex, chunkIndex)
				}
			}
		}
	}

	for i, chunk := range m.Chunks {
		if chunk.Index != i {
			return fmt.Errorf("chunk at position %d has index %d", i, chunk.Index)
		}
		if chunk.Image == "" || chunk.PathInImage == "" || chunk.SHA256 == "" || chunk.GroupTag == "" {
			return fmt.Errorf("chunk %d is missing required metadata", chunk.Index)
		}
		if chunk.Size <= 0 {
			return fmt.Errorf("chunk %d size must be greater than zero", chunk.Index)
		}
	}

	return nil
}

func Save(path string, m *Manifest) error {
	if err := m.Validate(); err != nil {
		return fmt.Errorf("validate manifest: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write manifest %q: %w", path, err)
	}

	return nil
}

func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %q: %w", path, err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %q: %w", path, err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("validate manifest %q: %w", path, err)
	}

	return &m, nil
}

func BuildImageReference(registry, repository, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registry, repository, tag)
}

func RewriteImageRegistry(imageRef, newRegistry string) (string, error) {
	if newRegistry == "" {
		return imageRef, nil
	}

	slash := -1
	for i := 0; i < len(imageRef); i++ {
		if imageRef[i] == '/' {
			slash = i
			break
		}
	}
	if slash <= 0 || slash == len(imageRef)-1 {
		return "", fmt.Errorf("invalid image reference %q", imageRef)
	}

	return fmt.Sprintf("%s/%s", newRegistry, imageRef[slash+1:]), nil
}
