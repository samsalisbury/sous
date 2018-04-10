package logging

type graphiteError struct {
	error
	CallerInfo
}

func reportGraphiteError(ls LogSink, err error) {
	msg := newGraphiteError(err)
	msg.ExcludeMe()
	Deliver(ls, msg)
}

func newGraphiteError(err error) graphiteError {
	return graphiteError{
		error:      err,
		CallerInfo: GetCallerInfo(NotHere()),
	}
}

func (ge graphiteError) DefaultLevel() Level {
	return WarningLevel
}

func (ge graphiteError) Message() string {
	return ge.Error()
}
