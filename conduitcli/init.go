package conduitcli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	cliutil "go.inout.gg/conduit/internal/cli"
	"go.inout.gg/foundations/must"
)

func newCommandInit() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initializes migration directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := filepath.Join(must.Must(os.Getwd()), DefaultMigrationDir)
			return cliutil.Init(cmd.Context(), dir, &cliutil.InitConfig{})
		},
	}
}
