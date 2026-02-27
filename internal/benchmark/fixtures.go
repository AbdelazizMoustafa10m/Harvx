//go:build bench

// Package benchmark provides synthetic repository generators and fixture helpers
// for performance benchmarking of Harvx pipeline stages. All functions in this
// package are gated behind the "bench" build tag and excluded from regular
// test runs.
//
// The package generates realistic file trees with appropriate content for each
// file type, supporting benchmarks from 1K to 50K files. Large fixture sets
// (50K files) are cached via sync.Once to avoid redundant regeneration across
// benchmark iterations.
package benchmark

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/harvx/harvx/internal/pipeline"
)

// extensionWeights defines the probability distribution of file extensions in
// generated repos. Weights are relative; higher weight means more files of that
// type. The distribution approximates a typical polyglot repository.
var extensionWeights = []struct {
	ext    string
	weight int
}{
	{".go", 30},
	{".ts", 20},
	{".py", 15},
	{".md", 10},
	{".json", 10},
	{".yaml", 8},
	{".toml", 7},
}

// totalWeight is the sum of all extension weights, computed once at package init.
var totalWeight int

func init() {
	for _, ew := range extensionWeights {
		totalWeight += ew.weight
	}
}

// directoryLayouts defines the directory structures used when generating repos.
// Files are distributed across flat directories, moderate depth (3 levels), and
// deep nesting (10 levels) to exercise different traversal patterns.
var directoryLayouts = []struct {
	path  string
	depth int
}{
	// Flat directories.
	{"src", 0},
	{"lib", 0},
	{"docs", 0},
	{"scripts", 0},

	// 3 levels deep.
	{"internal/pkg/core", 3},
	{"internal/pkg/util", 3},
	{"internal/api/handlers", 3},
	{"cmd/server/config", 3},

	// 10 levels deep.
	{"deep/a/b/c/d/e/f/g/h/i", 10},
	{"nested/level1/level2/level3/level4/level5/level6/level7/level8/level9", 10},
}

// cachedLargeRepo holds state for the sync.Once-cached 50K-file repository.
var (
	cachedLargeRepo     string
	cachedLargeRepoOnce sync.Once
	cachedLargeRepoErr  error
)

// GenerateTestRepo creates a synthetic repository under dir with the specified
// number of files. Files are distributed across directories of varying depth
// (flat, 3-level, 10-level) with realistic extensions (.go, .ts, .py, .md,
// .json, .yaml, .toml) and content. File sizes range from 100B to 50KB, with
// a small number of 1MB large files included to exercise --skip-large-files.
//
// The function creates the directory structure if it does not exist and fills
// files with content appropriate for each extension via GenerateRealisticContent.
// It calls tb.Fatal on any filesystem error, so callers never need to check
// for errors. The output is deterministic for a given fileCount (fixed seed).
func GenerateTestRepo(tb testing.TB, dir string, fileCount int) {
	tb.Helper()

	rng := rand.New(rand.NewSource(42)) //nolint:gosec // deterministic for reproducibility

	// Create all directory layouts.
	for _, layout := range directoryLayouts {
		dirPath := filepath.Join(dir, layout.path)
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			tb.Fatalf("creating directory %s: %v", dirPath, err)
		}
	}

	// Reserve slots for large files: 1 per 1000 files, minimum 1.
	largeFileCount := fileCount / 1000
	if largeFileCount < 1 {
		largeFileCount = 1
	}
	regularFileCount := fileCount - largeFileCount

	// Generate regular files distributed across directories.
	for i := range regularFileCount {
		layout := directoryLayouts[i%len(directoryLayouts)]
		ext := pickExtension(rng)
		size := 100 + rng.Intn(50*1024-100) // 100B to ~50KB
		name := fmt.Sprintf("file_%05d%s", i, ext)
		filePath := filepath.Join(dir, layout.path, name)

		content := GenerateRealisticContent(ext, size)
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			tb.Fatalf("writing file %s: %v", filePath, err)
		}
	}

	// Generate large files (1MB each) for --skip-large-files testing.
	for i := range largeFileCount {
		ext := pickExtension(rng)
		name := fmt.Sprintf("large_%03d%s", i, ext)
		filePath := filepath.Join(dir, "src", name)

		content := GenerateRealisticContent(ext, 1024*1024) // 1MB
		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			tb.Fatalf("writing large file %s: %v", filePath, err)
		}
	}
}

// GenerateCachedLargeRepo returns the path to a cached 50K-file repository,
// generating it only once per process using sync.Once. This avoids the
// multi-minute setup cost when running multiple benchmarks that need the
// large fixture set.
//
// The returned directory persists for the lifetime of the process. Cleanup is
// registered via tb.Cleanup so temporary files are removed when the test
// binary exits.
func GenerateCachedLargeRepo(tb testing.TB) string {
	tb.Helper()

	cachedLargeRepoOnce.Do(func() {
		dir, err := os.MkdirTemp("", "harvx-bench-50k-*")
		if err != nil {
			cachedLargeRepoErr = fmt.Errorf("creating temp dir for 50K repo: %w", err)
			return
		}
		cachedLargeRepo = dir

		// Use a shim TB that does not call Fatal (sync.Once must complete
		// without runtime.Goexit or the Once deadlocks).
		shimTB := &nonFatalTB{TB: tb}
		GenerateTestRepo(shimTB, dir, 50_000)
		if shimTB.failed {
			cachedLargeRepoErr = fmt.Errorf("generating 50K repo: %s", shimTB.msg)
		}
	})

	if cachedLargeRepoErr != nil {
		tb.Fatalf("cached large repo: %v", cachedLargeRepoErr)
	}

	tb.Cleanup(func() {
		// Cleanup is best-effort; the OS reclaims temp dirs eventually.
		_ = os.RemoveAll(cachedLargeRepo)
	})

	return cachedLargeRepo
}

// GenerateFileDescriptors creates a slice of FileDescriptor values with Content
// pre-loaded, suitable for benchmarking pipeline stages that operate on
// in-memory content without requiring files on disk. Each descriptor has a
// realistic relative path, size, language, and content matching its extension.
//
// Files are not written to disk. The dir parameter is used only to construct
// AbsPath values for descriptors that may reference filesystem paths. The
// output is deterministic for a given (dir, fileCount) pair.
func GenerateFileDescriptors(dir string, fileCount int) []pipeline.FileDescriptor {
	rng := rand.New(rand.NewSource(42)) //nolint:gosec // deterministic for reproducibility

	fds := make([]pipeline.FileDescriptor, 0, fileCount)

	for i := range fileCount {
		layout := directoryLayouts[i%len(directoryLayouts)]
		ext := pickExtension(rng)
		size := 100 + rng.Intn(50*1024-100) // 100B to ~50KB
		name := fmt.Sprintf("file_%05d%s", i, ext)
		relPath := filepath.Join(layout.path, name)
		absPath := filepath.Join(dir, relPath)

		content := GenerateRealisticContent(ext, size)

		fds = append(fds, pipeline.FileDescriptor{
			Path:     relPath,
			AbsPath:  absPath,
			Size:     int64(len(content)),
			Tier:     pipeline.DefaultTier,
			Content:  content,
			Language: extToLanguage(ext),
		})
	}

	return fds
}

// GenerateRealisticContent produces synthetic source content appropriate for the
// given file extension. The content is deterministic for a given (ext, size)
// pair and contains realistic structural elements: package declarations,
// imports, function definitions, class definitions, docstrings, headers,
// nested objects, and configuration blocks. The returned string is
// approximately size bytes long (truncated if generated content exceeds size).
func GenerateRealisticContent(ext string, size int) string {
	if size <= 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(size)

	switch ext {
	case ".go":
		writeGoContent(&b, size)
	case ".ts":
		writeTypeScriptContent(&b, size)
	case ".py":
		writePythonContent(&b, size)
	case ".md":
		writeMarkdownContent(&b, size)
	case ".json":
		writeJSONContent(&b, size)
	case ".yaml":
		writeYAMLContent(&b, size)
	case ".toml":
		writeTOMLContent(&b, size)
	default:
		writeGoContent(&b, size)
	}

	result := b.String()
	if len(result) > size {
		return result[:size]
	}
	return result
}

// pickExtension selects a random file extension based on the weighted
// distribution defined in extensionWeights.
func pickExtension(rng *rand.Rand) string {
	n := rng.Intn(totalWeight)
	cumulative := 0
	for _, ew := range extensionWeights {
		cumulative += ew.weight
		if n < cumulative {
			return ew.ext
		}
	}
	return ".go"
}

// extToLanguage maps a file extension to its programming language name for the
// FileDescriptor.Language field.
func extToLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts":
		return "typescript"
	case ".py":
		return "python"
	case ".md":
		return "markdown"
	case ".json":
		return "json"
	case ".yaml":
		return "yaml"
	case ".toml":
		return "toml"
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// Content generators
// ---------------------------------------------------------------------------
// Each generator writes repeating sections of realistic content for its file
// type, cycling through structural elements until the target size is reached.
// The content is intentionally verbose to produce realistic token counts.

// writeGoContent generates realistic Go source code into the builder.
func writeGoContent(b *strings.Builder, size int) {
	sections := []string{
		"package main\n\n",

		"import (\n\t\"context\"\n\t\"fmt\"\n\t\"log/slog\"\n\t\"os\"\n\t\"sync\"\n)\n\n",

		"// Config holds the application configuration.\ntype Config struct {\n\tHost    string `json:\"host\"`\n\tPort    int    `json:\"port\"`\n\tDebug   bool   `json:\"debug\"`\n\tTimeout int    `json:\"timeout\"`\n}\n\n",

		"// Server represents the main application server.\ntype Server struct {\n\tcfg     Config\n\tlogger  *slog.Logger\n\tmu      sync.Mutex\n\trunning bool\n}\n\n",

		"// NewServer creates a new Server with the given configuration.\nfunc NewServer(cfg Config) *Server {\n\treturn &Server{\n\t\tcfg:    cfg,\n\t\tlogger: slog.Default(),\n\t}\n}\n\n",

		"// Start begins accepting connections on the configured host and port.\nfunc (s *Server) Start(ctx context.Context) error {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\n\tif s.running {\n\t\treturn fmt.Errorf(\"server already running\")\n\t}\n\n\ts.logger.Info(\"starting server\",\n\t\t\"host\", s.cfg.Host,\n\t\t\"port\", s.cfg.Port,\n\t)\n\n\ts.running = true\n\treturn nil\n}\n\n",

		"// Stop gracefully shuts down the server.\nfunc (s *Server) Stop() error {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\n\tif !s.running {\n\t\treturn nil\n\t}\n\n\ts.logger.Info(\"stopping server\")\n\ts.running = false\n\treturn nil\n}\n\n",

		"// handleRequest processes an incoming request and returns the response body.\nfunc (s *Server) handleRequest(ctx context.Context, path string) (string, error) {\n\tselect {\n\tcase <-ctx.Done():\n\t\treturn \"\", ctx.Err()\n\tdefault:\n\t}\n\n\ts.logger.Debug(\"handling request\", \"path\", path)\n\n\tresult := fmt.Sprintf(\"response for %s\", path)\n\treturn result, nil\n}\n\n",

		"func main() {\n\tcfg := Config{\n\t\tHost:    \"localhost\",\n\t\tPort:    8080,\n\t\tDebug:   true,\n\t\tTimeout: 30,\n\t}\n\n\tsrv := NewServer(cfg)\n\tif err := srv.Start(context.Background()); err != nil {\n\t\tfmt.Fprintf(os.Stderr, \"failed to start: %v\\n\", err)\n\t\tos.Exit(1)\n\t}\n\tdefer srv.Stop()\n\n\tfmt.Println(\"server running\")\n}\n",
	}

	writeRepeatingContent(b, sections, size)
}

// writeTypeScriptContent generates realistic TypeScript source code into the builder.
func writeTypeScriptContent(b *strings.Builder, size int) {
	sections := []string{
		"import { EventEmitter } from 'events';\nimport { readFile, writeFile } from 'fs/promises';\nimport path from 'path';\n\n",

		"export interface Config {\n  host: string;\n  port: number;\n  debug: boolean;\n  timeout: number;\n  retries: number;\n}\n\n",

		"export interface RequestOptions {\n  method: 'GET' | 'POST' | 'PUT' | 'DELETE';\n  headers: Record<string, string>;\n  body?: unknown;\n  signal?: AbortSignal;\n}\n\n",

		"export class HttpClient extends EventEmitter {\n  private readonly baseUrl: string;\n  private readonly timeout: number;\n\n  constructor(config: Config) {\n    super();\n    this.baseUrl = `http://${config.host}:${config.port}`;\n    this.timeout = config.timeout;\n  }\n\n",

		"  async request<T>(path: string, options: RequestOptions): Promise<T> {\n    const url = new URL(path, this.baseUrl);\n    const controller = new AbortController();\n    const timer = setTimeout(() => controller.abort(), this.timeout);\n\n    try {\n      const response = await fetch(url.toString(), {\n        method: options.method,\n        headers: options.headers,\n        body: options.body ? JSON.stringify(options.body) : undefined,\n        signal: options.signal ?? controller.signal,\n      });\n\n      if (!response.ok) {\n        throw new Error(`HTTP ${response.status}: ${response.statusText}`);\n      }\n\n      return await response.json() as T;\n    } finally {\n      clearTimeout(timer);\n    }\n  }\n\n",

		"  async get<T>(path: string): Promise<T> {\n    return this.request<T>(path, {\n      method: 'GET',\n      headers: { 'Accept': 'application/json' },\n    });\n  }\n\n",

		"  async post<T>(path: string, body: unknown): Promise<T> {\n    return this.request<T>(path, {\n      method: 'POST',\n      headers: {\n        'Accept': 'application/json',\n        'Content-Type': 'application/json',\n      },\n      body,\n    });\n  }\n}\n\n",

		"export function createClient(config: Partial<Config> = {}): HttpClient {\n  const defaults: Config = {\n    host: 'localhost',\n    port: 3000,\n    debug: false,\n    timeout: 5000,\n    retries: 3,\n  };\n\n  return new HttpClient({ ...defaults, ...config });\n}\n\n",

		"export async function loadConfig(filePath: string): Promise<Config> {\n  const raw = await readFile(filePath, 'utf-8');\n  const parsed = JSON.parse(raw) as Partial<Config>;\n\n  if (!parsed.host || !parsed.port) {\n    throw new Error('Config must include host and port');\n  }\n\n  return parsed as Config;\n}\n",
	}

	writeRepeatingContent(b, sections, size)
}

// writePythonContent generates realistic Python source code into the builder.
func writePythonContent(b *strings.Builder, size int) {
	sections := []string{
		"\"\"\"Application server module.\n\nProvides the main Server class for handling HTTP requests\nwith configurable middleware and routing.\n\"\"\"\n\nimport asyncio\nimport logging\nfrom dataclasses import dataclass, field\nfrom typing import Any, Callable, Optional\n\n",

		"logger = logging.getLogger(__name__)\n\n\n",

		"@dataclass\nclass Config:\n    \"\"\"Server configuration.\"\"\"\n\n    host: str = \"localhost\"\n    port: int = 8080\n    debug: bool = False\n    workers: int = 4\n    timeout: float = 30.0\n    max_connections: int = 1000\n\n\n",

		"@dataclass\nclass Request:\n    \"\"\"Represents an incoming HTTP request.\"\"\"\n\n    method: str\n    path: str\n    headers: dict[str, str] = field(default_factory=dict)\n    body: Optional[bytes] = None\n    query_params: dict[str, str] = field(default_factory=dict)\n\n    @property\n    def content_type(self) -> Optional[str]:\n        \"\"\"Return the Content-Type header value.\"\"\"\n        return self.headers.get(\"content-type\")\n\n\n",

		"class Router:\n    \"\"\"URL router that maps paths to handler functions.\"\"\"\n\n    def __init__(self) -> None:\n        self._routes: dict[str, Callable] = {}\n        self._middleware: list[Callable] = []\n\n    def route(self, path: str) -> Callable:\n        \"\"\"Decorator to register a route handler.\n\n        Args:\n            path: The URL path pattern to match.\n\n        Returns:\n            Decorator function that registers the handler.\n        \"\"\"\n        def decorator(func: Callable) -> Callable:\n            self._routes[path] = func\n            return func\n        return decorator\n\n    def add_middleware(self, middleware: Callable) -> None:\n        \"\"\"Add a middleware function to the processing chain.\"\"\"\n        self._middleware.append(middleware)\n\n    async def dispatch(self, request: Request) -> Any:\n        \"\"\"Dispatch a request to the matching handler.\"\"\"\n        handler = self._routes.get(request.path)\n        if handler is None:\n            raise ValueError(f\"No handler for path: {request.path}\")\n        return await handler(request)\n\n\n",

		"class Server:\n    \"\"\"Async HTTP server with middleware support.\"\"\"\n\n    def __init__(self, config: Config) -> None:\n        self.config = config\n        self.router = Router()\n        self._running = False\n        self._connections: list[asyncio.StreamWriter] = []\n\n    async def start(self) -> None:\n        \"\"\"Start the server and begin accepting connections.\"\"\"\n        if self._running:\n            raise RuntimeError(\"Server is already running\")\n\n        logger.info(\n            \"Starting server on %s:%d\",\n            self.config.host,\n            self.config.port,\n        )\n        self._running = True\n\n    async def stop(self) -> None:\n        \"\"\"Gracefully stop the server.\"\"\"\n        if not self._running:\n            return\n\n        logger.info(\"Stopping server\")\n        for writer in self._connections:\n            writer.close()\n        self._running = False\n\n\n",

		"def create_app(config: Optional[Config] = None) -> Server:\n    \"\"\"Create and configure a new Server instance.\n\n    Args:\n        config: Optional server configuration. Uses defaults if not provided.\n\n    Returns:\n        A configured Server instance ready to start.\n    \"\"\"\n    if config is None:\n        config = Config()\n\n    server = Server(config)\n    return server\n\n\n",

		"async def main() -> None:\n    \"\"\"Entry point for the application.\"\"\"\n    config = Config(debug=True)\n    app = create_app(config)\n\n    try:\n        await app.start()\n        logger.info(\"Application running\")\n    except KeyboardInterrupt:\n        logger.info(\"Shutting down\")\n    finally:\n        await app.stop()\n\n\nif __name__ == \"__main__\":\n    asyncio.run(main())\n",
	}

	writeRepeatingContent(b, sections, size)
}

// writeMarkdownContent generates realistic Markdown documentation into the builder.
func writeMarkdownContent(b *strings.Builder, size int) {
	sections := []string{
		"# Project Documentation\n\nThis document provides an overview of the project architecture,\nsetup instructions, and development guidelines.\n\n",

		"## Getting Started\n\nFollow these steps to set up your development environment:\n\n1. Clone the repository\n2. Install dependencies\n3. Configure environment variables\n4. Run the development server\n\n",

		"### Prerequisites\n\n- Go 1.24 or later\n- Node.js 20+ (for frontend tooling)\n- Docker (optional, for containerized development)\n- PostgreSQL 16+ (or use the Docker Compose setup)\n\n",

		"## Architecture\n\nThe application follows a clean architecture pattern with clearly\nseparated layers:\n\n| Layer | Responsibility |\n|-------|---------------|\n| CLI | User interaction, argument parsing |\n| Service | Business logic, orchestration |\n| Repository | Data access, persistence |\n| Domain | Core types, validation rules |\n\n",

		"## Configuration\n\nConfiguration is loaded from multiple sources with the following\nprecedence (highest to lowest):\n\n1. Command-line flags\n2. Environment variables\n3. Configuration file (`.config.toml`)\n4. Default values\n\n```toml\n[server]\nhost = \"localhost\"\nport = 8080\n\n[database]\nurl = \"postgres://localhost:5432/app\"\npool_size = 10\n```\n\n",

		"## API Reference\n\n### Endpoints\n\n#### GET /api/v1/resources\n\nReturns a paginated list of resources.\n\n**Query Parameters:**\n\n| Parameter | Type | Default | Description |\n|-----------|------|---------|-------------|\n| page | int | 1 | Page number |\n| limit | int | 20 | Items per page |\n| sort | string | created_at | Sort field |\n\n**Response:**\n\n```json\n{\n  \"data\": [],\n  \"total\": 0,\n  \"page\": 1,\n  \"limit\": 20\n}\n```\n\n",

		"## Development\n\n### Running Tests\n\n```bash\ngo test ./...\ngo test -race ./...\ngo test -bench=. ./...\n```\n\n### Code Style\n\nWe follow standard Go conventions. Run `go vet` and `gofmt` before\nsubmitting pull requests.\n\n",

		"## Contributing\n\n1. Fork the repository\n2. Create a feature branch (`git checkout -b feature/amazing`)\n3. Commit your changes (`git commit -m 'Add amazing feature'`)\n4. Push to the branch (`git push origin feature/amazing`)\n5. Open a Pull Request\n\nPlease ensure all tests pass and add tests for new functionality.\n\n",

		"## License\n\nThis project is licensed under the MIT License. See the [LICENSE](LICENSE)\nfile for details.\n\n---\n\n*Generated documentation. Last updated automatically.*\n",
	}

	writeRepeatingContent(b, sections, size)
}

// writeJSONContent generates realistic nested JSON into the builder.
func writeJSONContent(b *strings.Builder, size int) {
	sections := []string{
		"{\n",
		"  \"name\": \"example-project\",\n  \"version\": \"2.1.0\",\n  \"description\": \"A sample project for testing and benchmarking\",\n",
		"  \"author\": {\n    \"name\": \"Test Author\",\n    \"email\": \"author@example.com\",\n    \"url\": \"https://example.com\"\n  },\n",
		"  \"repository\": {\n    \"type\": \"git\",\n    \"url\": \"https://github.com/example/project.git\"\n  },\n",
		"  \"dependencies\": {\n    \"express\": \"^4.18.2\",\n    \"cors\": \"^2.8.5\",\n    \"helmet\": \"^7.1.0\",\n    \"compression\": \"^1.7.4\",\n    \"dotenv\": \"^16.3.1\",\n    \"winston\": \"^3.11.0\",\n    \"joi\": \"^17.11.0\"\n  },\n",
		"  \"devDependencies\": {\n    \"typescript\": \"^5.3.3\",\n    \"jest\": \"^29.7.0\",\n    \"eslint\": \"^8.56.0\",\n    \"prettier\": \"^3.2.4\",\n    \"nodemon\": \"^3.0.2\"\n  },\n",
		"  \"scripts\": {\n    \"build\": \"tsc\",\n    \"start\": \"node dist/index.js\",\n    \"dev\": \"nodemon src/index.ts\",\n    \"test\": \"jest --coverage\",\n    \"lint\": \"eslint src/\",\n    \"format\": \"prettier --write src/\"\n  },\n",
		"  \"config\": {\n    \"server\": {\n      \"host\": \"0.0.0.0\",\n      \"port\": 3000,\n      \"cors\": {\n        \"origin\": [\"http://localhost:3000\", \"https://app.example.com\"],\n        \"credentials\": true\n      }\n    },\n",
		"    \"database\": {\n      \"host\": \"localhost\",\n      \"port\": 5432,\n      \"name\": \"appdb\",\n      \"pool\": {\n        \"min\": 2,\n        \"max\": 10,\n        \"idle_timeout\": 30000\n      }\n    },\n",
		"    \"logging\": {\n      \"level\": \"info\",\n      \"format\": \"json\",\n      \"transports\": [\"console\", \"file\"]\n    }\n  },\n",
		"  \"engines\": {\n    \"node\": \">=20.0.0\",\n    \"npm\": \">=10.0.0\"\n  },\n",
		"  \"keywords\": [\"api\", \"server\", \"rest\", \"typescript\", \"testing\"],\n  \"license\": \"MIT\",\n  \"private\": false\n",
		"}\n",
	}

	// JSON is structured, so write the base sections first, then pad with
	// repeated nested entries if more space is needed.
	for _, section := range sections {
		if b.Len() >= size {
			break
		}
		b.WriteString(section)
	}

	filler := "  \"item_%d\": {\n    \"id\": %d,\n    \"name\": \"entry-%d\",\n    \"value\": %d,\n    \"active\": true\n  },\n"
	i := 0
	for b.Len() < size {
		b.WriteString(fmt.Sprintf(filler, i, i, i, i*100))
		i++
	}
}

// writeYAMLContent generates realistic YAML configuration into the builder.
func writeYAMLContent(b *strings.Builder, size int) {
	sections := []string{
		"# Application configuration\n---\n\n",

		"server:\n  host: localhost\n  port: 8080\n  read_timeout: 30s\n  write_timeout: 30s\n  max_header_bytes: 1048576\n  tls:\n    enabled: false\n    cert_file: /etc/ssl/certs/server.crt\n    key_file: /etc/ssl/private/server.key\n\n",

		"database:\n  driver: postgres\n  host: localhost\n  port: 5432\n  name: application\n  user: app_user\n  ssl_mode: disable\n  pool:\n    max_open: 25\n    max_idle: 10\n    max_lifetime: 5m\n  migrations:\n    auto_migrate: true\n    directory: ./migrations\n\n",

		"logging:\n  level: info\n  format: json\n  output: stdout\n  fields:\n    service: myapp\n    version: \"1.0.0\"\n    environment: development\n  file:\n    enabled: false\n    path: /var/log/app/app.log\n    max_size: 100\n    max_backups: 3\n    max_age: 28\n\n",

		"cache:\n  driver: redis\n  host: localhost\n  port: 6379\n  password: \"\"\n  db: 0\n  ttl: 1h\n  prefix: \"app:\"\n  pool_size: 10\n\n",

		"auth:\n  jwt:\n    secret_key: change-me-in-production\n    expiration: 24h\n    refresh_expiration: 168h\n    issuer: myapp\n  oauth:\n    github:\n      client_id: \"\"\n      client_secret: \"\"\n      redirect_url: http://localhost:8080/auth/callback\n    google:\n      client_id: \"\"\n      client_secret: \"\"\n      redirect_url: http://localhost:8080/auth/google/callback\n\n",

		"features:\n  rate_limiting:\n    enabled: true\n    requests_per_minute: 60\n    burst: 10\n  cors:\n    allowed_origins:\n      - http://localhost:3000\n      - https://app.example.com\n    allowed_methods:\n      - GET\n      - POST\n      - PUT\n      - DELETE\n    allowed_headers:\n      - Authorization\n      - Content-Type\n    max_age: 3600\n\n",

		"monitoring:\n  metrics:\n    enabled: true\n    port: 9090\n    path: /metrics\n  health:\n    enabled: true\n    path: /health\n    checks:\n      - name: database\n        timeout: 5s\n      - name: cache\n        timeout: 3s\n\n",
	}

	writeRepeatingContent(b, sections, size)
}

// writeTOMLContent generates realistic TOML configuration into the builder.
func writeTOMLContent(b *strings.Builder, size int) {
	sections := []string{
		"# Application Configuration\n\ntitle = \"Application\"\nversion = \"1.0.0\"\nenvironment = \"development\"\n\n",

		"[server]\nhost = \"localhost\"\nport = 8080\nread_timeout = \"30s\"\nwrite_timeout = \"30s\"\nmax_request_size = 10485760\ngraceful_shutdown = \"15s\"\n\n",

		"[server.tls]\nenabled = false\ncert_file = \"/etc/ssl/certs/server.crt\"\nkey_file = \"/etc/ssl/private/server.key\"\nmin_version = \"1.2\"\n\n",

		"[database]\ndriver = \"postgres\"\nhost = \"localhost\"\nport = 5432\nname = \"application\"\nuser = \"app_user\"\npassword = \"\"\nssl_mode = \"disable\"\n\n",

		"[database.pool]\nmax_open_conns = 25\nmax_idle_conns = 10\nconn_max_lifetime = \"5m\"\nconn_max_idle_time = \"1m\"\n\n",

		"[logging]\nlevel = \"info\"\nformat = \"json\"\noutput = \"stdout\"\nadd_source = true\n\n",

		"[logging.fields]\nservice = \"myapp\"\nversion = \"1.0.0\"\n\n",

		"[[profiles]]\nname = \"default\"\nmax_tokens = 128000\nformat = \"markdown\"\ntarget = \"generic\"\n\n",

		"[[profiles]]\nname = \"claude\"\nmax_tokens = 200000\nformat = \"xml\"\ntarget = \"claude\"\ncompression = true\n\n",

		"[security]\nenable_redaction = true\nfail_on_secrets = false\nentropy_threshold = 4.5\n\n[[security.patterns]]\nname = \"aws_key\"\npattern = \"AKIA[0-9A-Z]{16}\"\n\n[[security.patterns]]\nname = \"github_token\"\npattern = \"ghp_[A-Za-z0-9]{36}\"\n\n",

		"[discovery]\nrespect_gitignore = true\nskip_hidden = true\nmax_file_size = 1048576\nfollow_symlinks = false\n\n",

		"[discovery.exclude]\npatterns = [\n  \"**/node_modules/**\",\n  \"**/.git/**\",\n  \"**/vendor/**\",\n  \"**/dist/**\",\n  \"**/*.min.js\",\n  \"**/*.min.css\",\n]\n\n",
	}

	writeRepeatingContent(b, sections, size)
}

// writeRepeatingContent writes sections into the builder, cycling through them
// until the target size is reached. This produces content that is structurally
// realistic at every point, not just padded with whitespace.
func writeRepeatingContent(b *strings.Builder, sections []string, size int) {
	if len(sections) == 0 {
		return
	}

	i := 0
	for b.Len() < size {
		b.WriteString(sections[i%len(sections)])
		i++
	}
}

// nonFatalTB wraps a testing.TB to capture Fatal calls without terminating the
// goroutine. This is necessary when GenerateTestRepo is called inside
// sync.Once, because runtime.Goexit (called by Fatalf) inside sync.Once
// causes a deadlock.
type nonFatalTB struct {
	testing.TB
	failed bool
	msg    string
}

func (t *nonFatalTB) Fatalf(format string, args ...any) {
	t.failed = true
	t.msg = fmt.Sprintf(format, args...)
}

func (t *nonFatalTB) Fatal(args ...any) {
	t.failed = true
	t.msg = fmt.Sprint(args...)
}

func (t *nonFatalTB) Helper() {}
