package output

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden reports whether golden files should be regenerated instead of
// compared. Set HARVX_UPDATE_GOLDEN=1 in the environment to regenerate.
func updateGolden() bool {
	return os.Getenv("HARVX_UPDATE_GOLDEN") == "1"
}

// loadGoldenFile reads a golden file from the given path. If the file does not
// exist, the test is failed with a message suggesting to set HARVX_UPDATE_GOLDEN=1.
func loadGoldenFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden: read %s: %v (set HARVX_UPDATE_GOLDEN=1 to generate)", path, err)
	}
	return data
}

// compareGolden compares got against the golden file at goldenPath. When
// HARVX_UPDATE_GOLDEN=1 is set, it writes got to goldenPath instead of
// comparing, creating parent directories as needed. If the golden file does
// not exist and HARVX_UPDATE_GOLDEN is not set, the file is created
// automatically (first-run generation) and the test passes.
//
// When not updating and the file exists, a byte-for-byte mismatch causes the
// test to fail with both expected and actual content shown.
func compareGolden(t *testing.T, got []byte, goldenPath string) {
	t.Helper()

	if updateGolden() {
		writeGoldenFile(t, goldenPath, got)
		return
	}

	// If the golden file does not exist, create it automatically on first run.
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		writeGoldenFile(t, goldenPath, got)
		t.Logf("golden: created %s (first run)", goldenPath)
		return
	}

	expected := loadGoldenFile(t, goldenPath)

	if !bytes.Equal(got, expected) {
		// Find first difference for a more helpful message.
		maxShow := 500
		gotStr := string(got)
		expStr := string(expected)
		if len(gotStr) > maxShow {
			gotStr = gotStr[:maxShow] + "...(truncated)"
		}
		if len(expStr) > maxShow {
			expStr = expStr[:maxShow] + "...(truncated)"
		}

		t.Errorf("golden mismatch for %s\n--- expected (len=%d) ---\n%s\n--- actual (len=%d) ---\n%s",
			goldenPath, len(expected), expStr, len(got), gotStr)
	}
}

// writeGoldenFile writes data to goldenPath, creating parent directories as
// needed.
func writeGoldenFile(t *testing.T, goldenPath string, data []byte) {
	t.Helper()

	dir := filepath.Dir(goldenPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("golden: create dir %s: %v", dir, err)
	}
	if err := os.WriteFile(goldenPath, data, 0644); err != nil {
		t.Fatalf("golden: write %s: %v", goldenPath, err)
	}
	t.Logf("golden: updated %s", goldenPath)
}
