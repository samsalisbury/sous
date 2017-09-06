package logging

type logConfigurationWarning struct {
	CallerInfo
	CallTime
	message string
}

func reportLogConfigurationWarning(ls LogSet, msg string) {
	warning := newLogConfigurationWarning(msg)
	Deliver(warning, ls)
}

func newLogConfigurationWarning(msg string) *logConfigurationWarning {
	return &logConfigurationWarning{
		message:    msg,
		CallTime:   GetCallTime(),
		CallerInfo: GetCallerInfo("reportLogConfigurationWarning", "newLogConfigurationWarning"),
	}
}

func (l *logConfigurationWarning) DefaultLevel() Level {
	return WarningLevel
}

func (l *logConfigurationWarning) Message() string {
	return l.message
}

func (l *logConfigurationWarning) EachField(f FieldReportFn) {
	l.CallTime.EachField(f)
	l.CallerInfo.EachField(f)
}
