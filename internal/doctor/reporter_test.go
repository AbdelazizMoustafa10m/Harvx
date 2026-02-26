package doctor

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndicator(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPass, "[PASS]"},
		{StatusWarn, "[WARN]"},
		{StatusFail, "[FAIL]"},
		{Status("unknown"), "[????]"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.want, indicator(tt.status))
		})
	}
}

func TestFormatText(t *testing.T) {
	report := &DoctorReport{
		Directory: "/tmp/test",
		Timestamp: "2026-01-01T00:00:00Z",
		Checks: []CheckResult{
			{
				Name:    "Git Repository",
				Status:  StatusPass,
				Message: "Git repository on branch main",
				Details: []string{"Branch: main", "HEAD: abc1234"},
			},
			{
				Name:    "Large Binary Files",
				Status:  StatusWarn,
				Message: "Found 1 large binary file(s) >1MB",
				Details: []string{"big.bin"},
			},
			{
				Name:    "Configuration",
				Status:  StatusFail,
				Message: "Invalid config: parse error",
			},
		},
	}

	var buf bytes.Buffer
	FormatText(&buf, report)
	output := buf.String()

	// Verify header.
	assert.Contains(t, output, "Harvx Doctor")
	assert.Contains(t, output, "/tmp/test")

	// Verify check indicators.
	assert.Contains(t, output, "[PASS] Git Repository")
	assert.Contains(t, output, "[WARN] Large Binary Files")
	assert.Contains(t, output, "[FAIL] Configuration")

	// Verify details are printed.
	assert.Contains(t, output, "Branch: main")
	assert.Contains(t, output, "big.bin")

	// Verify summary line.
	assert.Contains(t, output, "1 passed, 1 warnings, 1 failures")
}

func TestFormatText_EmptyReport(t *testing.T) {
	report := &DoctorReport{
		Directory: "/tmp/empty",
		Timestamp: "2026-01-01T00:00:00Z",
		Checks:    []CheckResult{},
	}

	var buf bytes.Buffer
	FormatText(&buf, report)
	output := buf.String()

	assert.Contains(t, output, "Harvx Doctor")
	assert.Contains(t, output, "0 passed, 0 warnings, 0 failures")
}

func TestFormatJSON(t *testing.T) {
	report := &DoctorReport{
		Directory: "/tmp/test",
		Timestamp: "2026-01-01T00:00:00Z",
		Checks: []CheckResult{
			{
				Name:    "Git Repository",
				Status:  StatusPass,
				Message: "Git repository on branch main",
			},
		},
		HasFail: false,
		HasWarn: false,
	}

	var buf bytes.Buffer
	err := FormatJSON(&buf, report)
	require.NoError(t, err)

	// Verify it's valid JSON.
	var decoded DoctorReport
	require.NoError(t, json.Unmarshal(buf.Bytes(), &decoded))

	assert.Equal(t, "/tmp/test", decoded.Directory)
	assert.Len(t, decoded.Checks, 1)
	assert.Equal(t, "Git Repository", decoded.Checks[0].Name)
	assert.Equal(t, StatusPass, decoded.Checks[0].Status)
}

func TestFormatJSON_Indented(t *testing.T) {
	report := &DoctorReport{
		Directory: "/tmp/test",
		Timestamp: "2026-01-01T00:00:00Z",
		Checks:    []CheckResult{},
	}

	var buf bytes.Buffer
	err := FormatJSON(&buf, report)
	require.NoError(t, err)

	// Verify the output is indented (contains newlines and spaces).
	assert.Contains(t, buf.String(), "\n")
	assert.Contains(t, buf.String(), "  ")
}

func TestFormatJSON_OmitsEmptyDetails(t *testing.T) {
	report := &DoctorReport{
		Directory: "/tmp/test",
		Timestamp: "2026-01-01T00:00:00Z",
		Checks: []CheckResult{
			{
				Name:    "Test",
				Status:  StatusPass,
				Message: "OK",
				// Details is nil -- should be omitted from JSON.
			},
		},
	}

	var buf bytes.Buffer
	err := FormatJSON(&buf, report)
	require.NoError(t, err)

	// The "details" key should not appear when nil (omitempty).
	assert.NotContains(t, buf.String(), `"details"`)
}
