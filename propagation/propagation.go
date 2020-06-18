package propagation

import (
	"fmt"
)

type propagationError struct {
	message      string
	wrappedError error
}

func (p *propagationError) Error() string {
	if p.wrappedError == nil {
		return p.message
	}
	return fmt.Sprintf(p.message, p.wrappedError)
}
