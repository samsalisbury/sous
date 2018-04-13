package logging

import (
	"bytes"
	"fmt"
)

type (
	resourceField float64
	//CPUResourceField is a specialization of float64 to handle details of ELK.
	CPUResourceField resourceField
	//MemResourceField is a specialization of float64 to handle details of ELK.
	MemResourceField resourceField
)

// EachField implements EachFielder on CPUResourceField.
func (f CPUResourceField) EachField(fn FieldReportFn) {
	fn(SousResourceCpus, resourceField(f))
}

// EachField implements EachFielder on CPUResourceField.
func (f MemResourceField) EachField(fn FieldReportFn) {
	fn(SousResourceMemory, resourceField(f))
}

// MarshalJSON implements json.Marshaller on CPUResourceField.
func (f resourceField) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%#g", f)
	return buf.Bytes(), nil
}
