package chunker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

type ChunkArtifact struct {
	Index       int
	FilePath    string
	FileName    string
	PathInImage string
	Size        int64
	SHA256      string
}

type GroupPlan struct {
	GroupIndex int
	Tag        string
	Image      string
	Chunks     []ChunkArtifact
}

type SplitResult struct {
	FileSize int64
	FileSHA  string
	Chunks   []ChunkArtifact
}

func SplitFile(ctx context.Context, sourcePath string, chunkSize int64, outputDir, imageChunkBaseDir string) (*SplitResult, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be greater than zero")
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("open source file %q: %w", sourcePath, err)
	}
	defer source.Close()

	fileInfo, err := source.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat source file %q: %w", sourcePath, err)
	}

	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("source file %q is empty", sourcePath)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create chunk output directory %q: %w", outputDir, err)
	}

	var (
		chunks     []ChunkArtifact
		totalBytes int64
		chunkIndex int
		fullHasher = sha256.New()
		readBuf    = make([]byte, 1024*1024)
	)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		chunkFileName := fmt.Sprintf("chunk-%05d.bin", chunkIndex)
		chunkPath := filepath.Join(outputDir, chunkFileName)
		dest, err := os.Create(chunkPath)
		if err != nil {
			return nil, fmt.Errorf("create chunk file %q: %w", chunkPath, err)
		}

		chunkHasher := sha256.New()
		limited := &io.LimitedReader{R: source, N: chunkSize}

		written, err := copyWithHashes(dest, limited, readBuf, fullHasher, chunkHasher)
		closeErr := dest.Close()
		if err != nil {
			return nil, fmt.Errorf("write chunk file %q: %w", chunkPath, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close chunk file %q: %w", chunkPath, closeErr)
		}
		if written == 0 {
			if removeErr := os.Remove(chunkPath); removeErr != nil {
				return nil, fmt.Errorf("remove empty chunk file %q: %w", chunkPath, removeErr)
			}
			break
		}

		chunks = append(chunks, ChunkArtifact{
			Index:       chunkIndex,
			FilePath:    chunkPath,
			FileName:    chunkFileName,
			PathInImage: path.Join(imageChunkBaseDir, chunkFileName),
			Size:        written,
			SHA256:      hex.EncodeToString(chunkHasher.Sum(nil)),
		})
		totalBytes += written
		chunkIndex++
	}

	return &SplitResult{
		FileSize: totalBytes,
		FileSHA:  hex.EncodeToString(fullHasher.Sum(nil)),
		Chunks:   chunks,
	}, nil
}

func GroupChunks(chunks []ChunkArtifact, chunksPerImage int, registry, repository, release string) []GroupPlan {
	var groups []GroupPlan
	if chunksPerImage < 1 {
		chunksPerImage = 1
	}

	for start := 0; start < len(chunks); start += chunksPerImage {
		end := start + chunksPerImage
		if end > len(chunks) {
			end = len(chunks)
		}

		groupIndex := len(groups)
		tag := imageTag(release, chunksPerImage, groupIndex, chunks[start].Index)
		image := fmt.Sprintf("%s/%s:%s", registry, repository, tag)

		groupChunks := make([]ChunkArtifact, end-start)
		copy(groupChunks, chunks[start:end])

		groups = append(groups, GroupPlan{
			GroupIndex: groupIndex,
			Tag:        tag,
			Image:      image,
			Chunks:     groupChunks,
		})
	}

	return groups
}

func imageTag(release string, chunksPerImage, groupIndex, chunkIndex int) string {
	if chunksPerImage <= 1 {
		return fmt.Sprintf("%s-%05d", release, chunkIndex)
	}
	return fmt.Sprintf("%s-g%05d", release, groupIndex)
}

func AssembleFile(destination string, chunkPaths []string) error {
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create output file %q: %w", destination, err)
	}
	defer out.Close()

	buffer := make([]byte, 1024*1024)
	for _, chunkPath := range chunkPaths {
		in, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("open chunk %q: %w", chunkPath, err)
		}

		if _, err := io.CopyBuffer(out, in, buffer); err != nil {
			in.Close()
			return fmt.Errorf("append chunk %q: %w", chunkPath, err)
		}
		if err := in.Close(); err != nil {
			return fmt.Errorf("close chunk %q: %w", chunkPath, err)
		}
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("sync output file %q: %w", destination, err)
	}

	return nil
}

func copyWithHashes(dst io.Writer, src io.Reader, buf []byte, hashWriters ...io.Writer) (int64, error) {
	var written int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			chunk := buf[:nr]
			for _, h := range hashWriters {
				if _, err := h.Write(chunk); err != nil {
					return written, err
				}
			}
			nw, ew := dst.Write(chunk)
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nw != nr {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er == io.EOF {
				return written, nil
			}
			return written, er
		}
	}
}
