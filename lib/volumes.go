package sous

import "fmt"

type (
	// Volume describes a deployment's volume mapping
	Volume struct {
		Host, Container string
		Mode            VolumeMode
	}

	// Volumes represents a list of volume mappings
	Volumes []*Volume

	//VolumeMode is either readwrite or readonly
	VolumeMode string
)

const (
	// ReadOnly specifies that a volume can only be read
	ReadOnly VolumeMode = "RO"
	// ReadWrite specifies that the container can write to the volume
	ReadWrite VolumeMode = "RW"
)

// Equal is used to compare Volumes pairs
func (vs Volumes) Equal(o Volumes) bool {
	if len(vs) != len(o) {
		Log.Vomit.Print("Volume lengths differ")
		return false
	}
	c := append(Volumes{}, o...)
	for _, v := range vs {
		m := false
		for i, ov := range c {
			if v.Equal(ov) {
				m = true
				if i < len(c) {
					c[i] = c[len(c)-1]
				}
				c = c[:len(c)-1]
				break
			}
		}
		if !m {
			Log.Vomit.Printf("missing volume: %v", v)
			return false
		}
	}
	if len(c) == 0 {
		return true
	}
	Log.Vomit.Printf("missing volumes: %v", c)
	return false
}

// Equal is used to compare *Volume pairs
func (v *Volume) Equal(o *Volume) bool {
	return v.Host == o.Host && v.Container == o.Container && v.Mode == o.Mode
}

func (vs Volumes) String() string {
	res := "["
	for _, v := range vs {
		res += fmt.Sprintf("%v,", v)
	}
	return res + "]"
}
