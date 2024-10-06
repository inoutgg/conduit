package apply

import (
	"github.com/urfave/cli/v2"
	"go.inout.gg/conduit"
	"go.inout.gg/conduit/internal/command/common"
	"go.inout.gg/conduit/internal/direction"
)

var stepsFlagName = "steps"

func NewCommand(migrator conduit.Migrator) *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Args:  true,
		Usage: "apply migrations in the given direction",
		Flags: []cli.Flag{
			common.DatabaseURLFlag,
			&cli.IntFlag{
				Name:  stepsFlagName,
				Usage: "maximum migrations steps",
			},
		},
		Action: func(ctx *cli.Context) error {
			return apply(ctx, migrator)
		},
	}
}

// apply applies a migration in the defined direction.
func apply(
	ctx *cli.Context,
	migrator conduit.Migrator,
) error {
	dir, err := direction.FromString(ctx.Args().First())
	if err != nil {
		return err
	}

	conn, err := common.Conn(ctx)
	if err != nil {
		return err
	}

	opts := &conduit.MigrateOptions{
		Steps: ctx.Int(stepsFlagName),
	}
	_, err = migrator.Migrate(ctx.Context, dir, conn, opts)
	if err != nil {
		return err
	}

	return nil
}
