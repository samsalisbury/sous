package logging

import (
	"bytes"
	"fmt"
)

type (
	//CPUResourceField is a specialization of float64 to handle details of ELK.
	CPUResourceField float64
	//MemResourceField is a specialization of float64 to handle details of ELK.
	MemResourceField int64
)

// EachField implements EachFielder on CPUResourceField.
func (f CPUResourceField) EachField(fn FieldReportFn) {
	fn(SousResourceCpus, f)
}

// EachField implements EachFielder on CPUResourceField.
func (f MemResourceField) EachField(fn FieldReportFn) {
	fn(SousResourceMemory, f)
}

// MarshalJSON implements json.Marshaller on CPUResourceField.
func (f CPUResourceField) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "%#g", f)
	return buf.Bytes(), nil
}
