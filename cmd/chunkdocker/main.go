package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/saeid/chunkdocker/internal/config"
	"github.com/saeid/chunkdocker/internal/fetcher"
	"github.com/saeid/chunkdocker/internal/logging"
	"github.com/saeid/chunkdocker/internal/publisher"
)

const defaultConfigPath = "config.yaml"

func main() {
	logger := logging.New(os.Stdout)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error

	switch os.Args[1] {
	case "publish":
		err = runPublish(ctx, logger, os.Args[2:])
	case "fetch":
		err = runFetch(ctx, logger, os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}

	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

		logger.Errorf("error: %v", err)
		os.Exit(1)
	}
}

func runPublish(ctx context.Context, logger *logging.Logger, args []string) error {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	configPath := fs.String("config", defaultConfigPath, "path to YAML config")
	sourceFile := fs.String("file", "", "path to source file")
	manifestOut := fs.String("manifest-out", "", "optional manifest output path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sourceFile == "" {
		return errors.New("publish requires --file")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	return publisher.New(cfg, logger).Publish(ctx, publisher.Request{
		SourceFile:      *sourceFile,
		ManifestOutPath: *manifestOut,
	})
}

func runFetch(ctx context.Context, logger *logging.Logger, args []string) error {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	configPath := fs.String("config", defaultConfigPath, "path to YAML config")
	manifestPath := fs.String("manifest", "", "path to manifest JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *manifestPath == "" {
		return errors.New("fetch requires --manifest")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	return fetcher.New(cfg, logger).Fetch(ctx, fetcher.Request{
		ManifestPath: *manifestPath,
	})
}

func printUsage() {
	fmt.Fprintf(os.Stdout, `chunkdocker transfers large files through Docker registries by chunking them into small images.

Usage:
  chunkdocker publish --file /path/to/file [--config config.yaml] [--manifest-out manifest.json]
  chunkdocker fetch --manifest /path/to/manifest.json [--config config.yaml]

Commands:
  publish    Chunk a source file, build/push Docker images, and write a manifest
  fetch      Pull grouped images, extract chunks, verify checksums, and restore the file

Flags:
  --config   Optional config path. Defaults to config.yaml in the current directory.
`)
}
