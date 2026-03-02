package commandutil

import "github.com/urfave/cli/v3"

func StringOr(cmd *cli.Command, name, fallback string) string {
	if cmd.IsSet(name) {
		return cmd.String(name)
	}

	return fallback
}

func BoolOr(cmd *cli.Command, name string, fallback bool) bool {
	if cmd.IsSet(name) {
		return cmd.Bool(name)
	}

	return fallback
}

func StringSliceOr(cmd *cli.Command, name string, fallback []string) []string {
	if cmd.IsSet(name) {
		return cmd.StringSlice(name)
	}

	return fallback
}
