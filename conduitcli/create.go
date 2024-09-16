package conduitcli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	cliutil "go.inout.gg/conduit/internal/cli"
	"go.inout.gg/foundations/must"
)

func newCommandCreate() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new migration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("conduit: missing `name\" argument")
			}

			name := args[0]
			dir := filepath.Join(must.Must(os.Getwd()), DefaultMigrationDir)
			version := time.Now().UnixMilli()

			return cliutil.CreateMigrationFile(cmd.Context(), dir, version, name, "sql")
		},
	}
}
