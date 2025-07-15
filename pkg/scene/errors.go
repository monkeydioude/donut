package scene

import "errors"

var (
	ErrCanNotGoBackState = errors.New("can not go back to previous state")
	ErrInvalidNextState  = errors.New("can not change state")
)
