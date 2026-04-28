package scheduledsdk

import "errors"

var (
	ErrNilClient   = errors.New("scheduledsdk: nil client")
	ErrNilAdapter  = errors.New("scheduledsdk: nil adapter")
	ErrUnsupported = errors.New("scheduledsdk: unsupported operation")
)
