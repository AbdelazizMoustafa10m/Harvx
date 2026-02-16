package cli

import (
	"encoding/json"
	"fmt"

	"github.com/harvx/harvx/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version and build information",
	Long:  "Display the harvx version, git commit, build date, Go version, and OS/architecture.",
	RunE:  runVersion,
}

func init() {
	versionCmd.Flags().Bool("json", false, "output version info as JSON")
	rootCmd.AddCommand(versionCmd)
}

// versionInfo holds structured version data for JSON output.
type versionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

func runVersion(cmd *cobra.Command, args []string) error {
	info := versionInfo{
		Version:   buildinfo.Version,
		Commit:    buildinfo.Commit,
		Date:      buildinfo.Date,
		GoVersion: buildinfo.GoVersion,
		OS:        buildinfo.OS(),
		Arch:      buildinfo.Arch(),
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "harvx version %s\n", info.Version)
	fmt.Fprintf(cmd.OutOrStdout(), "  commit:     %s\n", info.Commit)
	fmt.Fprintf(cmd.OutOrStdout(), "  built:      %s\n", info.Date)
	fmt.Fprintf(cmd.OutOrStdout(), "  go version: %s\n", info.GoVersion)
	fmt.Fprintf(cmd.OutOrStdout(), "  os/arch:    %s/%s\n", info.OS, info.Arch)

	return nil
}
