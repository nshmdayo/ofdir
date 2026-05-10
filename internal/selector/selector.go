package selector

import (
	"errors"
	"os/exec"

	"github.com/nshmdayo/opfd/internal/config"
)

// ErrCancelled is returned when the user cancels the selection (e.g. Ctrl+C).
var ErrCancelled = errors.New("selection cancelled")

// Selector is the interface for interactive directory selection.
type Selector interface {
	Select(candidates []string, prompt string) (string, error)
}

// New returns the best available Selector based on config.
func New(cfg *config.Config) Selector {
	switch cfg.UI.FuzzyFinder {
	case "fzf":
		if commandExists("fzf") {
			return &FzfSelector{}
		}
		return &InternalSelector{}
	case "peco":
		if commandExists("peco") {
			return &PecoSelector{}
		}
		return &InternalSelector{}
	default:
		return &InternalSelector{}
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
