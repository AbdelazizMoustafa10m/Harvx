// Package testutil provides shared test helpers for the harvx test suite.
// Helpers in this package are intended to be used from *_test.go files across
// all internal packages.
package testutil

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// update is a package-level flag that controls whether golden files are
// regenerated instead of compared. Pass -update on the test binary command
// line to regenerate all golden files in one pass:
//
//	go test ./... -update
var update = flag.Bool("update", false, "regenerate golden files")

// Golden compares actual against the golden file stored at
// testdata/golden/<name>.golden relative to the calling test's working
// directory.
//
// When the -update flag is set, Golden writes actual to the golden file and
// returns immediately without failing the test. This allows intentional output
// changes to be committed in a single pass.
//
// When -update is not set, Golden reads the golden file and compares it
// byte-for-byte against actual. Any mismatch causes the test to fail with a
// diff-style message showing both the expected and actual content.
//
// The golden file directory is created automatically when -update is set.
func Golden(t *testing.T, name string, actual []byte) {
	t.Helper()

	golden := filepath.Join("testdata", "golden", name+".golden")

	if *update {
		dir := filepath.Dir(golden)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("golden: create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(golden, actual, 0644); err != nil {
			t.Fatalf("golden: write %s: %v", golden, err)
		}
		return
	}

	expected, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden: read %s: %v (run with -update to generate)", golden, err)
	}

	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch for %s\n--- expected\n%s\n--- actual\n%s",
			name, expected, actual)
	}
}
