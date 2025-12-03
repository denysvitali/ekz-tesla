package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/denysvitali/ekz-tesla/cmd/root"
)

// Build information. Populated at build-time via ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	Date      = "unknown"
	GoVersion = runtime.Version()
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, commit hash, build date, and Go version.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ekz-tesla %s\n", Version)
		fmt.Printf("  Commit:     %s\n", Commit)
		fmt.Printf("  Built:      %s\n", Date)
		fmt.Printf("  Go version: %s\n", GoVersion)
		fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	root.RootCmd.AddCommand(VersionCmd)
}
