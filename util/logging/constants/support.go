package constants

// EachField implements logging.EachFielder on OTLName.
// This means you can hand the FieldReportFn to an OTLName and not worry about it.
func (o OTLName) EachField(fn func(FieldName, interface{})) {
	fn(Loglov3Otl, o)
}
