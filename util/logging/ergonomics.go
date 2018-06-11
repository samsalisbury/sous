package logging

// logging.Deliver(dsm.log, logging.DebugLevel, logging.GetCallerInfo(), logging.SousGenericV1,

// Debug sends a generic-otl Debug log message.
func Debug(l LogSink, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, DebugLevel, GetCallerInfo(NotHere())}, fs...)...)
}

// Info sends a generic-otl Info log message.
func Info(l LogSink, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, InformationLevel, GetCallerInfo(NotHere())}, fs...)...)
}

// Warn sends a generic-otl Warn log message.
func Warn(l LogSink, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, WarningLevel, GetCallerInfo(NotHere())}, fs...)...)
}

// DebugMsg sends a generic-otl Debug log message, wrapping a string message.
func DebugMsg(l LogSink, msg string, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, DebugLevel, GetCallerInfo(NotHere()), MessageField(msg)}, fs...)...)
}

// InfoMsg sends a generic-otl Info log message, wrapping a string message.
func InfoMsg(l LogSink, msg string, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, InformationLevel, GetCallerInfo(NotHere()), MessageField(msg)}, fs...)...)
}

// WarnMsg sends a generic-otl Warn log message, wrapping a string message.
func WarnMsg(l LogSink, msg string, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, WarningLevel, GetCallerInfo(NotHere()), MessageField(msg)}, fs...)...)
}

// DebugConsole sends a generic-otl Debug log message, and echoes it to the console, wrapping a string message.
func DebugConsole(l LogSink, msg string, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, DebugLevel, GetCallerInfo(NotHere()), ConsoleAndMessage(msg)}, fs...)...)
}

// InfoConsole sends a generic-otl Info log message, and echoes it to the console, wrapping a string message.
func InfoConsole(l LogSink, msg string, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, InformationLevel, GetCallerInfo(NotHere()), ConsoleAndMessage(msg)}, fs...)...)
}

// WarnConsole sends a generic-otl Warn log message, and echoes it to the console, wrapping a string message.
func WarnConsole(l LogSink, msg string, fs ...interface{}) {
	Deliver(l, append([]interface{}{SousGenericV1, WarningLevel, GetCallerInfo(NotHere()), ConsoleAndMessage(msg)}, fs...)...)
}
