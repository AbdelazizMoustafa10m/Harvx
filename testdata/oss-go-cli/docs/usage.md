# Usage Guide

## Basic Synchronization

Sync files from a source directory to a destination:

```bash
gosync sync ./local-files /backup/files
```

## Watch Mode

Continuously watch for changes and sync automatically:

```bash
gosync sync --watch ./src ./backup
```

Press `Ctrl+C` to stop watching.

## Ignore Patterns

Skip files matching certain patterns:

```bash
gosync sync --ignore "*.log" --ignore "tmp/" ./src ./dst
```

## Checksum Verification

By default, gosync uses SHA-256 checksums to detect changes.
Disable checksums for faster syncs when timestamp comparison is sufficient:

```toml
[sync]
checksum = false
```

## Troubleshooting

Enable verbose logging to diagnose issues:

```bash
gosync --verbose sync ./src ./dst
```