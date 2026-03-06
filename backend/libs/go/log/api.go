package log

type Logger interface {
	Infof(format string, args ...any)
	Warn(error)
	Error(error)
}

type Stub struct{}

func (Stub) Infof(string, ...any) {}
func (Stub) Warn(error)           {}
func (Stub) Error(error)          {}
