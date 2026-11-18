package shortcut

import (
	"fmt"
	"os/exec"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

type ExistsFunc func(name string) bool

func LookPathExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func Collision(cfg config.Config, id string, exists ExistsFunc) error {
	if err := ValidateID(id); err != nil {
		return err
	}
	if _, ok := cfg.Shortcuts[id]; ok {
		return fmt.Errorf("%s is already a kpod shortcut", CommandName(id))
	}
	name := CommandName(id)
	if exists == nil {
		exists = LookPathExists
	}
	if exists(name) {
		return fmt.Errorf("%s already exists in your shell.\n\nChoose another shortcut.", name)
	}
	return nil
}
