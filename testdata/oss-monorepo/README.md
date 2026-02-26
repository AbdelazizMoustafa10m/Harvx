# Acme Platform

A monorepo containing the Acme Platform services: shared core library, REST API, web frontend, and background worker.

## Structure

```
packages/
  core/    - Shared TypeScript types and utilities
  api/     - Express.js REST API server
  web/     - React SPA frontend
services/
  worker/  - Go background job processor
scripts/
  build.sh - Cross-package build script
```

## Getting Started

### Prerequisites

- Node.js 20+
- Go 1.24+
- npm or pnpm

### Setup

```bash
# Install all JS dependencies
npm install --workspaces

# Build all packages
./scripts/build.sh
```

### Running Services

```bash
# API server
cd packages/api && npm run dev

# Web frontend
cd packages/web && npm run dev

# Worker service
cd services/worker && go run .
```

## Architecture

The platform follows a modular architecture:

- **core** provides shared types, validation, and utilities used by both api and web
- **api** serves the REST API and depends on core
- **web** is the React frontend consuming the API, depends on core for types
- **worker** is a standalone Go service processing background jobs from a queue

## License

Apache-2.0