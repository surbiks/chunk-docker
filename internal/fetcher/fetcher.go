package fetcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/saeid/chunkdocker/internal/chunker"
	"github.com/saeid/chunkdocker/internal/config"
	"github.com/saeid/chunkdocker/internal/dockerops"
	"github.com/saeid/chunkdocker/internal/filesystem"
	"github.com/saeid/chunkdocker/internal/hashutil"
	"github.com/saeid/chunkdocker/internal/logging"
	"github.com/saeid/chunkdocker/internal/manifest"
)

type Request struct {
	ManifestPath string
}

type Fetcher struct {
	cfg    *config.Root
	logger *logging.Logger
}

func New(cfg *config.Root, logger *logging.Logger) *Fetcher {
	return &Fetcher{cfg: cfg, logger: logger}
}

func (f *Fetcher) Fetch(ctx context.Context, req Request) error {
	clientCfg := f.cfg.Client

	if err := filesystem.EnsureDir(clientCfg.WorkDir); err != nil {
		return err
	}
	if err := filesystem.EnsureDir(clientCfg.TempDir); err != nil {
		return err
	}
	if err := filesystem.EnsureDir(clientCfg.OutputDir); err != nil {
		return err
	}

	docker, err := dockerops.New(clientCfg.DockerBinary, f.logger)
	if err != nil {
		return err
	}

	doc, err := manifest.Load(req.ManifestPath)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(clientCfg.OutputDir, doc.FileName)
	if !clientCfg.Assemble.Overwrite {
		if _, err := os.Stat(outputPath); err == nil {
			return fmt.Errorf("output file %q already exists and overwrite is false", outputPath)
		}
	}

	runDir, err := filesystem.CreateRunDir(clientCfg.TempDir, "fetch")
	if err != nil {
		return err
	}
	if !clientCfg.KeepTemp {
		defer func() {
			if cleanupErr := filesystem.RemoveAll(runDir); cleanupErr != nil {
				f.logger.Warnf("cleanup failed for %s: %v", runDir, cleanupErr)
			}
		}()
	}

	extractDir := filepath.Join(runDir, "extracted")
	if err := filesystem.EnsureDir(extractDir); err != nil {
		return err
	}

	groups := doc.Groups
	if len(groups) == 0 {
		groups = deriveGroups(doc)
	}

	chunkPaths := make([]string, len(doc.Chunks))
	for _, group := range groups {
		if err := f.pullAndExtractGroup(ctx, docker, doc, group, extractDir, chunkPaths); err != nil {
			return err
		}
	}

	f.logger.Infof("reassembling %d chunks into %s", len(chunkPaths), outputPath)
	if err := chunker.AssembleFile(outputPath, chunkPaths); err != nil {
		return err
	}

	outputHash, err := hashutil.FileSHA256(outputPath)
	if err != nil {
		return err
	}
	if outputHash != doc.FileSHA256 {
		return fmt.Errorf("final checksum mismatch for %s: got %s want %s", outputPath, outputHash, doc.FileSHA256)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("stat restored file %q: %w", outputPath, err)
	}
	if info.Size() != doc.FileSize {
		return fmt.Errorf("restored file size mismatch for %s: got %d want %d", outputPath, info.Size(), doc.FileSize)
	}

	f.logger.Infof("fetch completed successfully")
	return nil
}

func (f *Fetcher) pullAndExtractGroup(ctx context.Context, docker *dockerops.Client, doc *manifest.Manifest, group manifest.GroupEntry, extractDir string, chunkPaths []string) error {
	var image string
	if group.Image != "" {
		image = group.Image
	} else {
		return fmt.Errorf("manifest group %d is missing image reference", group.GroupIndex)
	}

	for attempt := 1; attempt <= f.cfg.Client.Download.Retries; attempt++ {
		if err := docker.Pull(ctx, image); err != nil {
			if attempt == f.cfg.Client.Download.Retries {
				return err
			}
			f.logger.Warnf("pull attempt %d/%d failed for %s: %v", attempt, f.cfg.Client.Download.Retries, image, err)
			continue
		}
		break
	}

	containerID, err := docker.CreateContainer(ctx, image)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := docker.RemoveContainer(ctx, containerID); cleanupErr != nil {
			f.logger.Warnf("container cleanup failed for %s: %v", containerID, cleanupErr)
		}
	}()

	for _, chunkIndex := range group.ChunkIndexes {
		chunk := doc.Chunks[chunkIndex]
		destPath := filepath.Join(extractDir, filepath.Base(chunk.PathInImage))

		f.logger.Infof("extracting chunk %d from %s", chunk.Index, image)
		if err := docker.CopyFromContainer(ctx, containerID, chunk.PathInImage, destPath); err != nil {
			return err
		}

		sum, err := hashutil.FileSHA256(destPath)
		if err != nil {
			return err
		}
		if sum != chunk.SHA256 {
			return fmt.Errorf("chunk checksum mismatch for chunk %d: got %s want %s", chunk.Index, sum, chunk.SHA256)
		}

		info, err := os.Stat(destPath)
		if err != nil {
			return fmt.Errorf("stat extracted chunk %q: %w", destPath, err)
		}
		if info.Size() != chunk.Size {
			return fmt.Errorf("chunk size mismatch for chunk %d: got %d want %d", chunk.Index, info.Size(), chunk.Size)
		}

		chunkPaths[chunk.Index] = destPath
	}

	return nil
}

func deriveGroups(doc *manifest.Manifest) []manifest.GroupEntry {
	indexed := make(map[int]*manifest.GroupEntry)
	order := make([]int, 0)
	for _, chunk := range doc.Chunks {
		entry, ok := indexed[chunk.GroupIndex]
		if !ok {
			entry = &manifest.GroupEntry{
				GroupIndex:   chunk.GroupIndex,
				Tag:          chunk.GroupTag,
				Image:        chunk.Image,
				ChunkIndexes: []int{},
			}
			indexed[chunk.GroupIndex] = entry
			order = append(order, chunk.GroupIndex)
		}
		entry.ChunkIndexes = append(entry.ChunkIndexes, chunk.Index)
	}

	result := make([]manifest.GroupEntry, 0, len(order))
	for _, idx := range order {
		result = append(result, *indexed[idx])
	}
	return result
}
