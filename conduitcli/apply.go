package conduitcli

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.inout.gg/conduit"
)

func newCommandApply(migrator conduit.Migrator) *cobra.Command {
	return &cobra.Command{
		Use:   "apply <direction>",
		Short: "Applies migrations in the direction",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("conduit: missing `name\" argument")
			}

			direction, err := stringToDirection(args[0])
			if err != nil {
				return err
			}

			// dir := filepath.Join(must.Must(os.Getwd()), DefaultMigrationDir)

			return migrator.Migrate(cmd.Context(), direction)
		},
	}
}

func stringToDirection(str string) (conduit.Direction, error) {
	switch str {
	case string(conduit.DirectionUp):
		return conduit.DirectionUp, nil
	case string(conduit.DirectionDown):
		return conduit.DirectionDown, nil
	}

	return "", fmt.Errorf("conduit: invalid direction, expected \"up\" or \"down\", recived: %s", str)
}
