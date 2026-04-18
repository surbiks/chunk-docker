# AGENTS.md

# Project Memory and Workflow Rules

This file is the working memory and execution guide for AI agents contributing to this project.
It must stay accurate, concise, and updated as the project evolves.

The project is a Go-based CLI tool for chunk-based file distribution through Docker registries.
Its purpose is to split large files into logical chunks, package them into Docker images, publish them to a registry, and later reconstruct the original file on a client in a restricted network.

---

## Core Workflow Rules (Always Follow)

When implementing a **new feature**, **major change**, **refactor**, or **architectural update**:

1. First understand the current project structure, tech stack, coding style, and constraints from:
   - this `AGENTS.md`
   - `README.md`
   - `go.mod`
   - the existing package structure
   - existing tests and config files

2. Before coding:
   - identify the smallest correct change that solves the task
   - state assumptions explicitly
   - check whether the request affects:
     - config schema
     - manifest schema
     - CLI behavior
     - Docker image layout
     - publish flow
     - fetch flow
     - tests
     - documentation

3. After completing the changes:
   - review what was added or modified
   - update this `AGENTS.md` if the change affects:
     - architecture
     - package responsibilities
     - config fields
     - manifest format
     - commands
     - development workflow
     - important implementation lessons
   - keep updates short, concrete, and useful
   - do not let this file become bloated

4. At the end of the task, provide a brief summary for the user covering:
   - what changed
   - what files were touched
   - what still remains or could be improved

Treat `AGENTS.md` as a **living project memory**.
It should help future sessions start with the correct context immediately.

---

## Project Purpose

This project builds a Go CLI application for transferring large files via Docker registries in restricted-network environments.

### Main idea
- On an external server:
  - take a file
  - split it into chunks
  - package chunks into Docker images
  - push images to a registry
  - generate a manifest

- On an internal/restricted client:
  - read the manifest
  - pull the required images
  - extract chunk files
  - reconstruct the original file
  - verify integrity with SHA256

### Important packaging modes
The system supports two packaging modes:

1. **Single-chunk image mode**
   - one logical chunk per image

2. **Multi-chunk image mode**
   - multiple logical chunks per image
   - number of chunks per image is configurable
   - each chunk must still be represented individually in the manifest
   - each chunk inside a grouped image should be added via a separate `COPY` instruction so it maps to a distinct Docker layer

This distinction is important and must not be lost during future refactors.

---

## Current MVP Architecture

- CLI entrypoint lives in `cmd/chunkdocker`.
- YAML config loading and human-readable byte-size parsing live in `internal/config`.
- Manifest JSON structs and read/write validation live in `internal/manifest`.
- Chunk splitting, grouping, and file assembly live in `internal/chunker`.
- Docker CLI wrappers live in `internal/dockerops`.
- Publish orchestration lives in `internal/publisher`.
- Fetch orchestration lives in `internal/fetcher`.
- Hash helpers live in `internal/hashutil`.
- Directory and file helpers live in `internal/filesystem`.
- Logging lives in `internal/logging`.

### Current command surface
- `publish --file ... [--config ...] [--manifest-out ...]`
- `fetch --manifest ... [--config ...]`

### Current config model
- The config is YAML with exactly two top-level sections:
  - `server`
  - `client`
- `server.chunk_size` supports values like `20MB` and `32MiB`.
- `server.chunks_per_image` controls grouping.
- `server.registry` is the push target.
- `server.manifest_registry` controls which registry is written into manifest image references and defaults to `server.registry`.
- `server.image_chunk_base_dir` must stay absolute and deterministic.
- `client.registry_override` can rewrite manifest image references at fetch time.
- `client.assemble.overwrite` controls whether fetch can replace an existing output file.

### Current manifest contract
- Manifest version is `1`.
- Manifest stores both `publish_registry` and `registry`.
- Chunks are always first-class and listed individually.
- Group entries list the chunk indexes each image contains.
- Each chunk entry includes:
  - `index`
  - `group_index`
  - `group_tag`
  - `image`
  - `path_in_image`
  - `size`
  - `sha256`

### Current implementation notes
- Docker integration is intentionally CLI-based through `os/exec`.
- Grouped images use `FROM scratch` plus one `COPY` per chunk.
- Docker builds disable provenance and SBOM attestations because some registry mirrors fail to fetch `application/vnd.in-toto+json` referrers during pull.
- Publish computes per-chunk SHA256 and full-file SHA256 while streaming the source file once.
- Fetch pulls each image once, extracts only the chunks listed for that image, verifies chunk checksums, reassembles in chunk order, then verifies the final file checksum.

---

## Tech Stack

- Language: Go
- Primary interface: CLI application
- Config format: YAML
- Manifest format: JSON
- Container interaction: Docker CLI via `os/exec`
- Main integrity algorithm: SHA256

### Preferred implementation style
- Prefer Go standard library where practical
- Keep dependencies minimal
- Prefer explicit, readable code over abstraction-heavy designs
- Use streaming IO for large files
- Avoid loading full large files into memory
- Keep Linux as the primary target for MVP unless the task explicitly expands platform support

---

## Behavioral Guidelines

These rules exist to reduce common AI coding mistakes.

### 1. Think Before Coding

Do not assume silently.

Before implementation:
- state assumptions clearly
- identify ambiguity
- surface tradeoffs when relevant
- do not invent requirements
- if multiple reasonable interpretations exist, mention them instead of silently choosing one

If a simpler solution is enough, prefer it.

### 2. Simplicity First

Write the minimum correct code for the requested task.

Avoid:
- speculative features
- future-proof abstractions with no current need
- configuration options that were not requested
- unnecessary interfaces
- unnecessary indirection
- overengineering for hypothetical future use cases

Ask:
- Is this the smallest correct change?
- Would a senior Go engineer consider this too complex?
- Can this be made clearer with fewer moving parts?

### 3. Surgical Changes

Touch only what is necessary.

When editing existing code:
- do not refactor unrelated areas
- do not reformat unrelated code
- do not rename things just for preference
- match the existing style unless the task requires changing it

You may remove code only when:
- your change makes it unused or incorrect
- it is directly related to the requested task

If you notice unrelated problems, mention them separately instead of fixing them unasked.

### 4. Goal-Driven Execution

Turn requests into verifiable goals.

Examples:
- "Fix fetch corruption bug" -> reproduce, fix, verify checksum passes
- "Add config option" -> parse config, validate it, test it
- "Add grouped-image support" -> generate grouped Dockerfiles, update manifest, fetch by group, add tests

For multi-step work, think in terms of:
1. implement
2. verify
3. document

Success must be testable, not vague.

---

## Project Architecture Expectations

Keep the codebase modular and responsibility-driven.

Expected package areas include concepts like:
- `cmd/...` for CLI entrypoint
- `internal/config` for YAML loading and validation
- `internal/manifest` for manifest structs and encoding/decoding
- `internal/chunker` for splitting and assembly logic
- `internal/dockerops` for Docker CLI wrappers
- `internal/publisher` for publish orchestration
- `internal/fetcher` for fetch orchestration
- `internal/hashutil` for SHA256 helpers
- optional helpers for filesystem and logging

### Responsibility boundaries
- Config code should not contain business workflow orchestration
- Docker code should wrap Docker operations, not own publish/fetch logic
- Manifest code should define and serialize metadata, not perform Docker operations
- Chunking code should deal with chunk split/assembly mechanics
- Publisher/fetcher should orchestrate the workflow using lower-level packages

Avoid package responsibility leakage.

---

## Critical Domain Rules

These are project-specific rules and must be preserved.

### 1. Logical chunks are first-class entities
Even when multiple chunks are stored in a single Docker image:
- chunks remain individually tracked
- chunks remain individually checksummed
- chunks remain individually ordered
- manifest entries must remain chunk-granular

### 2. Grouped images must not merge chunks into one blob
If `chunks_per_image > 1`, do **not**:
- concatenate chunks into one packed file before image build
- hide chunk boundaries inside a single image artifact
- represent multiple chunks as one manifest item

Instead:
- group consecutive chunks into one image
- add each chunk via a separate `COPY`
- keep deterministic chunk paths inside the image

### 3. Client reconstruction must not guess
The manifest must contain enough information for the client to reconstruct the file without inferring undocumented rules.

### 4. Integrity is mandatory
Both of these are required:
- per-chunk SHA256
- final reconstructed file SHA256

Never weaken integrity verification without explicit request.

### 5. Large-file friendliness matters
This project exists for constrained environments.
Always prefer implementations that:
- stream data
- reduce unnecessary disk/memory pressure
- keep Docker artifacts predictable
- preserve recoverability and debuggability

---

## Config Rules

The application is config-first.

### General expectations
- Prefer config file updates over adding new CLI flags
- New behavior should usually be controlled through config unless it is clearly a one-off command input
- Keep the CLI small and predictable

### Config structure
The config has two main top-level sections:
- `server`
- `client`

If a change affects configuration:
- update config structs
- update validation
- update sample config
- update README
- update this file if the config model changes materially

### Common config examples
Possible config concepts include:
- chunk size
- chunks per image
- work/temp directories
- docker binary path
- registry/repository/release values
- output directory
- retries
- overwrite behavior
- keep-temp behavior

Do not add config options casually.
Only add them when they solve a real need.

---

## Manifest Rules

The manifest is a contract between publisher and fetcher.

If a task changes manifest behavior:
- preserve backward clarity where possible
- update manifest structs
- update serialization/deserialization
- update README and examples
- update tests
- update this file if manifest semantics changed

The manifest should clearly describe:
- original file metadata
- chunk metadata
- grouping/image metadata
- image references
- deterministic chunk paths
- checksums

Do not introduce hidden reconstruction rules outside the manifest unless clearly documented and intentional.

---

## Docker Interaction Rules

For MVP and normal implementation:
- prefer Docker CLI via `os/exec`
- keep commands explicit and debuggable
- avoid shell-dependent behavior when not necessary

Typical operations may include:
- `docker build`
- `docker push`
- `docker pull`
- `docker create`
- `docker cp`
- `docker rm`

When changing Docker behavior:
- preserve deterministic image naming
- preserve deterministic chunk paths
- preserve minimal image construction
- keep logging clear enough for debugging

Do not replace CLI-based Docker integration with a more complex approach unless explicitly justified.

---

## Logging and Error Handling Rules

### Logging
Logs should be:
- concise
- useful
- progress-oriented
- human-readable

Good logs typically indicate:
- current file
- current chunk or group
- image tag being built/pulled
- verification progress
- final success/failure

### Errors
Errors should be:
- explicit
- contextual
- actionable where possible

Prefer errors like:
- "failed to push image docker.io/user/repo:v1-g00002: ..."
instead of:
- "push failed"

Cleanup failures should be logged, but should not hide the primary failure.

---

## Testing Rules

When practical, add or update tests for:
- config parsing and validation
- size parsing
- manifest encode/decode
- chunk metadata/grouping helpers
- hash helpers

When fixing a bug:
- first try to add a test that reproduces it
- then implement the fix

When adding a feature:
- add focused tests for the new behavior where reasonable

Keep tests targeted and maintainable.

---

## Documentation Rules

If the task changes user-facing behavior, update:
- `README.md`
- sample `config.yaml`
- manifest examples if relevant
- `AGENTS.md` if architecture, workflow, or conventions changed

Documentation updates should be concise and accurate.
Do not leave docs stale after changing behavior.

---

## Code Review Checklist for AI Agents

Before finishing, quickly verify:

1. Does the code solve only the requested problem?
2. Is the solution simpler than or equal to the obvious implementation?
3. Were unrelated files avoided?
4. Are config, manifest, and README updated if needed?
5. Are chunk/group semantics still correct?
6. Is integrity verification preserved?
7. Are errors contextual and understandable?
8. Are tests added or updated where practical?
9. Does `AGENTS.md` still reflect reality?

---

## Preferred Change Summary Format

At the end of significant work, summarize briefly in this format:

- What changed:
- Why it changed:
- Files touched:
- Config/manifest impact:
- Anything follow-up worth considering:

---

## Notes for Future Sessions

When starting a new task:
1. read this file first
2. inspect current project tree
3. inspect config and manifest models
4. identify whether the task affects:
   - chunking
   - grouping
   - Docker layering
   - publish flow
   - fetch flow
   - integrity checks
5. make the smallest correct change
6. update this file if project memory has changed
