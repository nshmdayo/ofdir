package cli

import (
	"fmt"

	"github.com/nshmdayo/ofdir/shell"
)

// PrintInitScript writes the shell integration script to stdout.
func PrintInitScript(sh string) error {
	switch sh {
	case "bash":
		fmt.Print(shell.BashInit)
	case "zsh":
		fmt.Print(shell.ZshInit)
	default:
		return errorf("unsupported shell %q (supported: bash, zsh)", sh)
	}
	return nil
}
