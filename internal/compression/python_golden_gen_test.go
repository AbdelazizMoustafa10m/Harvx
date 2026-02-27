package compression

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPythonCompressor_UpdateGolden regenerates .expected files for Python
// golden test fixtures. Run with:
//
//	GENERATE_GOLDEN=python go test -run TestPythonCompressor_UpdateGolden -v ./internal/compression/
func TestPythonCompressor_UpdateGolden(t *testing.T) {
	if os.Getenv("GENERATE_GOLDEN") != "python" {
		t.Skip("set GENERATE_GOLDEN=python to regenerate golden files")
	}

	compressor := NewPythonCompressor()
	ctx := context.Background()

	fixtures := []string{
		"python/django_model.py",
		"python/fastapi_router.py",
		"python/dataclass_types.py",
		"python/protocol_types.py",
		"python/decorators.py",
		"python/docstrings.py",
		"python/complete_file.py",
		"python/async_functions.py",
	}

	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			source := readFixture(t, fixture)
			output, err := compressor.Compress(ctx, source)
			if err != nil {
				t.Fatalf("compress error: %v", err)
			}
			rendered := strings.TrimSpace(output.Render())
			expectedPath := filepath.Join(testdataDir(), fixture+".expected")
			err = os.WriteFile(expectedPath, []byte(rendered+"\n"), 0644)
			if err != nil {
				t.Fatalf("write expected: %v", err)
			}
			t.Logf("wrote %s (%d bytes)", expectedPath, len(rendered))
		})
	}
}
