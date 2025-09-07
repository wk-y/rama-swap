package ramalama

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
)

type Ramalama struct {
	Command []string
}

type Model struct {
	Name     string
	Modified string
	Size     int
}

func (c Ramalama) GetModels() ([]Model, error) {
	if len(c.Command) == 0 {
		return nil, fmt.Errorf("invalid config: Ramalama Command should not be empty")
	}

	cliArgs := slices.Concat(c.Command[1:], []string{"list", "--json"})

	cmd := exec.Command(c.Command[0], cliArgs...)

	cmd.Stderr = nil
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to pipe command: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	var models []Model
	if err := json.NewDecoder(pipe).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to parse ramalama model list: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("ramalama error: %v", err)
	}

	return models, nil
}
