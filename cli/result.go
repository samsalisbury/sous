package cli

import (
	"encoding/base64"
	"fmt"
	"unicode/utf8"
)

type (
	// Result is a result of a CLI invokation.
	Result interface {
		// ExitCode is the exit code the program should exit with based on this
		// result.
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

func (s SuccessResult) String() string {
	if utf8.Valid(s.Data) {
		return string(s.Data)
	}
	return base64.StdEncoding.EncodeToString(s.Data)
}

func Success(v ...interface{}) SuccessResult {
	return SuccessResult{Data: []byte(fmt.Sprint(v...))}
}

func SuccessData(d []byte) SuccessResult { return SuccessResult{Data: d} }

func Successf(format string, v ...interface{}) Result {
	return SuccessResult{Data: []byte(fmt.Sprintf(format, v...))}
}
