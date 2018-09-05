package servermanager

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Model ...
type Model struct {
	binPth string
	envs   []string
}

func (model Model) IsBooted() (bool, error) {

	devBootCmd := command.New("/bin/ps", "aux")
	devBootOut, err := devBootCmd.RunAndReturnTrimmedCombinedOutput()
	log.Infof(devBootOut)
	if err != nil {
		return false, err
	}

	return strings.Contains(devBootOut, "MockRails"), nil
}

func serverBinPth() (string, error) {
	binPth := filepath.Join("/", "usr", "local", "bin", "rails")

	if exist, err := pathutil.IsPathExists(binPth); err != nil {
		return "", err
	} else if !exist {
		message := "no server installed"

		return "", fmt.Errorf("%s (%s)", message, binPth)
	}
	return binPth, nil
}

// New ...
func New() (*Model, error) {

	binPth, err := serverBinPth()
	if err != nil {
		return nil, err
	}

	envs := []string{}

	return &Model{
		binPth: binPth,
		envs:   envs,
	}, nil
}

// StartEmulatorCommand ...
func (model Model) StartServerCommand(options ...string) *command.Model {
	args := []string{model.binPth, "server"}
	args = append(args, "-b", "0.0.0.0")

	commandModel := command.New(args[0], args[1:]...).AppendEnvs(model.envs...)

	return commandModel
}
