package kindacool

import "errors"

var ErrInvalidConfig = errors.New("invalid config")

type GlobalOptions struct {
	Verbose bool
	Name    string
}

func (o *GlobalOptions) Validate() error {
	// TODO: validate name length
	return nil
}
