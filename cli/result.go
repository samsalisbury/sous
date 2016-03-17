package cli

import "fmt"

type (
	Result interface {
		ExitCode() int
	}
	Tipper interface {
		UserTip() string
	}
	// SuccessResult is a successful result.
	SuccessResult struct {
		// Data is the real return value of this function, it will be printed to
		// stdout by default, for consumption by other commands/pipelines etc.
		Data []byte
	}
)

func (s SuccessResult) ExitCode() int { return EX_OK }

func Success(v ...interface{}) SuccessResult {
	return SuccessResult{Data: []byte(fmt.Sprint(v...))}
}

func SuccessData(d []byte) SuccessResult { return SuccessResult{Data: d} }

func Successf(format string, v ...interface{}) Result {
	return SuccessResult{Data: []byte(fmt.Sprintf(format, v...))}
}
