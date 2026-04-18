# chunkdocker

`chunkdocker` is a Go CLI for moving large files through Docker registries by splitting them into logical chunks, storing those chunks in small `FROM scratch` images, and reconstructing the original file on the other side.

## What it does

- `publish` runs on an external server.
- It splits a large file into fixed-size chunks.
- It groups consecutive chunks into Docker images based on `chunks_per_image`.
- Each chunk is added to the image with its own `COPY` instruction, so grouped images still keep chunk boundaries visible as distinct layers.
- It pushes the images and writes a JSON manifest.
- The manifest can point to a different registry than the one used for push, which is useful when clients pull through a mirror or proxy registry.

- `fetch` runs inside the restricted network.
- It reads the manifest.
- It pulls each image once.
- It extracts the chunk files with `docker create` and `docker cp`.
- It verifies per-chunk SHA256 and final-file SHA256.
- It reassembles the original file in order.

## Project layout

```text
cmd/chunkdocker         CLI entrypoint
internal/config         YAML config loading, size parsing, validation
internal/manifest       Manifest structs and JSON helpers
internal/chunker        Chunk splitting, grouping, and assembly helpers
internal/dockerops      Docker CLI wrapper
internal/publisher      Publish workflow orchestration
internal/fetcher        Fetch workflow orchestration
internal/hashutil       SHA256 helpers
internal/filesystem     Directory and file helper functions
internal/logging        Small logger wrapper
```

## Build

```bash
go build ./cmd/chunkdocker
```

## Config

The CLI defaults to `config.yaml` in the current directory.

The config file must contain exactly two top-level sections:

- `server`
- `client`

See [config.yaml](config.yaml) for a sample.

Registry behavior:

- `server.registry`: where `publish` pushes images.
- `server.manifest_registry`: registry hostname written into manifest image references. If omitted, it defaults to `server.registry`.
- `client.registry_override`: optional fetch-time registry rewrite. If set, `fetch` rewrites manifest image references to use this registry.

## Usage

Publish a file:

```bash
./chunkdocker publish --file /data/bigfile.iso
```

Publish and choose the manifest path explicitly:

```bash
./chunkdocker publish --file /data/bigfile.iso --manifest-out ./bigfile.manifest.json
```

Fetch a file from a manifest:

```bash
./chunkdocker fetch --manifest ./bigfile.manifest.json
```

Use a non-default config file:

```bash
./chunkdocker publish --config ./config.yaml --file /data/bigfile.iso
./chunkdocker fetch --config ./config.yaml --manifest ./bigfile.manifest.json
```

## Manifest shape

The manifest describes both logical chunks and image groups. Each chunk entry records:

- global chunk index
- group index and tag
- full image reference
- deterministic path inside the image
- size
- SHA256

Example:

```json
{
  "version": 1,
  "created_at": "2026-04-18T10:00:00Z",
  "file_name": "ubuntu.iso",
  "file_size": 123456789,
  "file_sha256": "abc123",
  "chunk_size": 52428800,
  "chunk_count": 12,
  "chunks_per_image": 5,
  "publish_registry": "docker.io",
  "registry": "docker.io",
  "repository": "myuser/file-chunks",
  "release": "v1",
  "image_chunk_base_dir": "/chunks",
  "groups": [
    {
      "group_index": 0,
      "tag": "v1-g00000",
      "image": "docker.io/myuser/file-chunks:v1-g00000",
      "chunk_indexes": [0, 1, 2, 3, 4]
    }
  ],
  "chunks": [
    {
      "index": 0,
      "group_index": 0,
      "group_tag": "v1-g00000",
      "image": "docker.io/myuser/file-chunks:v1-g00000",
      "path_in_image": "/chunks/chunk-00000.bin",
      "size": 52428800,
      "sha256": "deadbeef"
    }
  ]
}
```

## How grouped images work

If `chunks_per_image = 1`, each chunk gets its own tag:

- `docker.io/myuser/file-chunks:v1-00000`
- `docker.io/myuser/file-chunks:v1-00001`

If `chunks_per_image > 1`, consecutive chunks are grouped:

- `docker.io/myuser/file-chunks:v1-g00000`
- `docker.io/myuser/file-chunks:v1-g00001`

Each grouped image still uses a Dockerfile like:

```dockerfile
FROM scratch
COPY chunk-00000.bin /chunks/chunk-00000.bin
COPY chunk-00001.bin /chunks/chunk-00001.bin
COPY chunk-00002.bin /chunks/chunk-00002.bin
```

## Notes

- This MVP is Linux-first.
- Docker integration uses the Docker CLI through `os/exec`.
- Because transport images use `FROM scratch`, fetch creates containers with a dummy command so `docker create` succeeds before `docker cp`; the command is never started.
- Image builds explicitly disable provenance and SBOM attestations so registry mirrors that do not support OCI referrers can still pull the images reliably.
- Manifest image references may intentionally differ from the push registry so restricted clients can pull from a mirror registry.
- The fetch workflow is intentionally sequential for predictable behavior.
- Cleanup failures are logged and do not mask the primary failure.

## Tests

```bash
go test ./...
```
