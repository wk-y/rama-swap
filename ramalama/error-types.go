package ramalama

type ErrEmptyCommand struct{}

// Error implements error.
func (e ErrEmptyCommand) Error() string {
	return "ramalama command should not be empty"
}

var _ error = ErrEmptyCommand{}
