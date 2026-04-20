package publisher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/saeid/chunkdocker/internal/chunker"
	"github.com/saeid/chunkdocker/internal/config"
	"github.com/saeid/chunkdocker/internal/dockerops"
	"github.com/saeid/chunkdocker/internal/filesystem"
	"github.com/saeid/chunkdocker/internal/logging"
	"github.com/saeid/chunkdocker/internal/manifest"
)

type Request struct {
	SourceFile      string
	ManifestOutPath string
}

type Publisher struct {
	cfg    *config.Root
	logger *logging.Logger
}

func New(cfg *config.Root, logger *logging.Logger) *Publisher {
	return &Publisher{cfg: cfg, logger: logger}
}

func (p *Publisher) Publish(ctx context.Context, req Request) error {
	serverCfg := p.cfg.Server

	info, err := os.Stat(req.SourceFile)
	if err != nil {
		return fmt.Errorf("stat input file %q: %w", req.SourceFile, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("input file %q is not a regular file", req.SourceFile)
	}

	if err := filesystem.EnsureDir(serverCfg.WorkDir); err != nil {
		return err
	}
	if err := filesystem.EnsureDir(serverCfg.TempDir); err != nil {
		return err
	}

	docker, err := dockerops.New(serverCfg.DockerBinary, p.logger)
	if err != nil {
		return err
	}

	runDir, err := filesystem.CreateRunDir(serverCfg.TempDir, "publish")
	if err != nil {
		return err
	}
	if !serverCfg.KeepTemp {
		defer func() {
			if cleanupErr := filesystem.RemoveAll(runDir); cleanupErr != nil {
				p.logger.Warnf("cleanup failed for %s: %v", runDir, cleanupErr)
			}
		}()
	}

	chunkDir := filepath.Join(runDir, "chunks")
	p.logger.Infof("splitting %s into chunks of %d bytes", req.SourceFile, serverCfg.ChunkSize.Int64())
	splitResult, err := chunker.SplitFile(ctx, req.SourceFile, serverCfg.ChunkSize.Int64(), chunkDir, serverCfg.ImageChunkBaseDir)
	if err != nil {
		return err
	}

	groups := chunker.GroupChunks(splitResult.Chunks, serverCfg.ChunksPerImage, serverCfg.Registry, serverCfg.Repository, serverCfg.Release)
	p.logger.Infof("prepared %d chunks across %d image groups", len(splitResult.Chunks), len(groups))

	for _, group := range groups {
		if err := p.buildAndPushGroup(ctx, docker, runDir, group); err != nil {
			return err
		}
	}

	manifestPath := req.ManifestOutPath
	if manifestPath == "" {
		manifestPath = config.DefaultManifestPath(req.SourceFile)
	}

	manifestDoc := buildManifest(req.SourceFile, splitResult, groups, serverCfg)
	p.logger.Infof("writing manifest to %s", manifestPath)
	if err := manifest.Save(manifestPath, manifestDoc); err != nil {
		return err
	}

	p.logger.Infof("publish completed successfully")
	return nil
}

func (p *Publisher) buildAndPushGroup(ctx context.Context, docker *dockerops.Client, runDir string, group chunker.GroupPlan) error {
	buildContext := filepath.Join(runDir, fmt.Sprintf("build-g%05d", group.GroupIndex))
	if err := filesystem.EnsureDir(buildContext); err != nil {
		return err
	}

	for _, chunk := range group.Chunks {
		targetPath := filepath.Join(buildContext, chunk.FileName)
		if err := filesystem.LinkOrCopy(chunk.FilePath, targetPath); err != nil {
			return fmt.Errorf("prepare build context for chunk %d: %w", chunk.Index, err)
		}
	}

	dockerfileLines := renderDockerfile(p.cfg.Server.Build.BaseImage, group.Chunks)

	dockerfilePath := filepath.Join(buildContext, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(strings.Join(dockerfileLines, "\n")+"\n"), 0o644); err != nil {
		return fmt.Errorf("write Dockerfile for group %d: %w", group.GroupIndex, err)
	}

	if err := docker.Build(ctx, dockerops.BuildOptions{
		ContextDir: buildContext,
		Dockerfile: dockerfilePath,
		Image:      group.Image,
		NoCache:    p.cfg.Server.Build.NoCache,
		Pull:       p.cfg.Server.Build.Pull,
	}); err != nil {
		return err
	}

	for attempt := 1; attempt <= p.cfg.Server.Push.Retries; attempt++ {
		if err := docker.Push(ctx, group.Image); err != nil {
			if attempt == p.cfg.Server.Push.Retries {
				return err
			}
			p.logger.Warnf("push attempt %d/%d failed for %s: %v", attempt, p.cfg.Server.Push.Retries, group.Image, err)
			continue
		}
		return nil
	}

	return nil
}

func renderDockerfile(baseImage string, chunks []chunker.ChunkArtifact) []string {
	lines := []string{fmt.Sprintf("FROM %s", baseImage)}
	for _, chunk := range chunks {
		lines = append(lines, fmt.Sprintf("COPY %s %s", chunk.FileName, chunk.PathInImage))
	}
	return lines
}

func buildManifest(sourceFile string, split *chunker.SplitResult, groups []chunker.GroupPlan, cfg config.ServerConfig) *manifest.Manifest {
	doc := &manifest.Manifest{
		Version:           manifest.CurrentVersion,
		CreatedAt:         time.Now().UTC(),
		FileName:          filepath.Base(sourceFile),
		FileSize:          split.FileSize,
		FileSHA256:        split.FileSHA,
		ChunkSize:         cfg.ChunkSize.Int64(),
		ChunkCount:        len(split.Chunks),
		ChunksPerImage:    cfg.ChunksPerImage,
		PublishRegistry:   cfg.Registry,
		Registry:          cfg.Registry,
		Repository:        cfg.Repository,
		Release:           cfg.Release,
		ImageChunkBaseDir: cfg.ImageChunkBaseDir,
		Groups:            make([]manifest.GroupEntry, 0, len(groups)),
		Chunks:            make([]manifest.ChunkEntry, 0, len(split.Chunks)),
	}

	for _, group := range groups {
		manifestImage := manifest.BuildImageReference(cfg.ManifestRegistry, cfg.Repository, group.Tag)
		groupEntry := manifest.GroupEntry{
			GroupIndex:   group.GroupIndex,
			Tag:          group.Tag,
			Image:        manifestImage,
			ChunkIndexes: make([]int, 0, len(group.Chunks)),
		}
		for _, chunk := range group.Chunks {
			groupEntry.ChunkIndexes = append(groupEntry.ChunkIndexes, chunk.Index)
			doc.Chunks = append(doc.Chunks, manifest.ChunkEntry{
				Index:       chunk.Index,
				GroupIndex:  group.GroupIndex,
				GroupTag:    group.Tag,
				Image:       manifestImage,
				PathInImage: chunk.PathInImage,
				Size:        chunk.Size,
				SHA256:      chunk.SHA256,
			})
		}
		doc.Groups = append(doc.Groups, groupEntry)
	}

	doc.Registry = cfg.ManifestRegistry

	return doc
}
