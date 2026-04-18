package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Root struct {
	Server ServerConfig `yaml:"server"`
	Client ClientConfig `yaml:"client"`
}

type ServerConfig struct {
	WorkDir           string   `yaml:"work_dir"`
	TempDir           string   `yaml:"temp_dir"`
	ChunkSize         ByteSize `yaml:"chunk_size"`
	ChunksPerImage    int      `yaml:"chunks_per_image"`
	DockerBinary      string   `yaml:"docker_binary"`
	Registry          string   `yaml:"registry"`
	ManifestRegistry  string   `yaml:"manifest_registry"`
	Repository        string   `yaml:"repository"`
	Release           string   `yaml:"release"`
	ImageChunkBaseDir string   `yaml:"image_chunk_base_dir"`
	KeepTemp          bool     `yaml:"keep_temp"`
	Build             Build    `yaml:"build"`
	Push              Push     `yaml:"push"`
}

type Build struct {
	NoCache bool `yaml:"no_cache"`
	Pull    bool `yaml:"pull"`
}

type Push struct {
	Retries int `yaml:"retries"`
}

type ClientConfig struct {
	WorkDir          string   `yaml:"work_dir"`
	TempDir          string   `yaml:"temp_dir"`
	OutputDir        string   `yaml:"output_dir"`
	DockerBinary     string   `yaml:"docker_binary"`
	RegistryOverride string   `yaml:"registry_override"`
	KeepTemp         bool     `yaml:"keep_temp"`
	Download         Download `yaml:"download"`
	Assemble         Assemble `yaml:"assemble"`
}

type Download struct {
	Parallelism int `yaml:"parallelism"`
	Retries     int `yaml:"retries"`
}

type Assemble struct {
	Overwrite bool `yaml:"overwrite"`
}

func Load(path string) (*Root, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Root
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %q: %w", path, err)
	}

	return &cfg, nil
}

func (c *Root) applyDefaults() {
	if c.Server.DockerBinary == "" {
		c.Server.DockerBinary = "docker"
	}
	if c.Server.ChunksPerImage == 0 {
		c.Server.ChunksPerImage = 1
	}
	if c.Server.ManifestRegistry == "" {
		c.Server.ManifestRegistry = c.Server.Registry
	}
	if c.Server.ImageChunkBaseDir == "" {
		c.Server.ImageChunkBaseDir = "/chunks"
	}
	if c.Server.Push.Retries == 0 {
		c.Server.Push.Retries = 1
	}

	if c.Client.DockerBinary == "" {
		c.Client.DockerBinary = "docker"
	}
	if c.Client.Download.Parallelism == 0 {
		c.Client.Download.Parallelism = 1
	}
	if c.Client.Download.Retries == 0 {
		c.Client.Download.Retries = 1
	}
}

func (c *Root) Validate() error {
	var errs []error

	if err := c.Server.validate(); err != nil {
		errs = append(errs, fmt.Errorf("server: %w", err))
	}
	if err := c.Client.validate(); err != nil {
		errs = append(errs, fmt.Errorf("client: %w", err))
	}

	return errors.Join(errs...)
}

func (c ServerConfig) validate() error {
	var errs []error

	if c.WorkDir == "" {
		errs = append(errs, errors.New("work_dir is required"))
	}
	if c.TempDir == "" {
		errs = append(errs, errors.New("temp_dir is required"))
	}
	if c.ChunkSize.Int64() <= 0 {
		errs = append(errs, errors.New("chunk_size must be greater than zero"))
	}
	if c.ChunksPerImage <= 0 {
		errs = append(errs, errors.New("chunks_per_image must be at least 1"))
	}
	if c.Registry == "" {
		errs = append(errs, errors.New("registry is required"))
	}
	if c.Repository == "" {
		errs = append(errs, errors.New("repository is required"))
	}
	if c.Release == "" {
		errs = append(errs, errors.New("release is required"))
	}
	if c.ImageChunkBaseDir == "" {
		errs = append(errs, errors.New("image_chunk_base_dir is required"))
	} else if cleaned := path.Clean(c.ImageChunkBaseDir); cleaned == "." || cleaned == "/" && c.ImageChunkBaseDir != "/" {
		errs = append(errs, errors.New("image_chunk_base_dir must be a valid absolute path"))
	} else if !path.IsAbs(c.ImageChunkBaseDir) {
		errs = append(errs, errors.New("image_chunk_base_dir must be absolute"))
	}
	if c.Push.Retries < 1 {
		errs = append(errs, errors.New("push.retries must be at least 1"))
	}

	return errors.Join(errs...)
}

func (c ClientConfig) validate() error {
	var errs []error

	if c.WorkDir == "" {
		errs = append(errs, errors.New("work_dir is required"))
	}
	if c.TempDir == "" {
		errs = append(errs, errors.New("temp_dir is required"))
	}
	if c.OutputDir == "" {
		errs = append(errs, errors.New("output_dir is required"))
	}
	if c.Download.Parallelism < 1 {
		errs = append(errs, errors.New("download.parallelism must be at least 1"))
	}
	if c.Download.Retries < 1 {
		errs = append(errs, errors.New("download.retries must be at least 1"))
	}

	return errors.Join(errs...)
}

func DefaultManifestPath(sourceFile string) string {
	base := filepath.Base(sourceFile)
	return fmt.Sprintf("%s.manifest.json", base)
}
