package ramalama

import (
	"context"
	"fmt"
	"os/exec"
	"slices"
)

type ServeArgs struct {
	Model string // required
	Port  int
	Alias *string
}

func (c Ramalama) ServeCommand(ctx context.Context, args ServeArgs) *exec.Cmd {
	cliArgs := slices.Concat(c.Command[1:], []string{"serve", "--pull", "never"})

	if args.Alias != nil {
		cliArgs = append(cliArgs, "-n", *args.Alias)
	}

	cliArgs = append(cliArgs, "-p", fmt.Sprint(args.Port))

	cliArgs = append(cliArgs, args.Model)

	return exec.CommandContext(ctx, c.Command[0], cliArgs...)
}
