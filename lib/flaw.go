package sous

type (
	// A Flaw captures the digression from a validation rule
	Flaw interface {
		Repair() error
	}

	// Flawed covers types that can be validated and have flaws
	// Be kind to them, because aren't we all flawed somehow?
	Flawed interface {
		// Validate returns a list of flaws that enumerate problems with the Flawed
		Validate() []Flaw
	}
)

// RepairAll attempts to repair all the flaws in a slice, and returns errors
// and flaws when any of the flaws return errors
func RepairAll(in []Flaw) ([]Flaw, []error) {
	var fs []Flaw
	var es []error

	for _, f := range in {
		if e := f.Repair(); e != nil {
			es = append(es, e)
			fs = append(fs, f)
		}
	}
	return fs, es
}
