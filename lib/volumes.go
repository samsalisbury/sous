package sous

import (
	"fmt"
	"io"

	"github.com/opentable/sous/util/logging"
)

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
		reportDebugVolumeMessage("Volume lengths differ", o, vs)
		return false
	}
	c := append(Volumes{}, o...)
	reportDebugVolumeMessage("compairing:", c, vs)

	for _, v := range vs {
		m := false
		for i, ov := range c {
			reportDebugVolumeMessage("compairing:", append(Volumes{}, v), append(Volumes{}, ov))
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
			reportDebugVolumeMessage("missing volume:", append(Volumes{}, v), Volumes{})
			return false
		}
	}
	if len(c) == 0 {
		return true
	}
	reportDebugVolumeMessage("missing volume:", c, Volumes{})
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

type volumeMessage struct {
	logging.CallerInfo
	msg          string
	volumePairA  Volumes
	volumePairB  Volumes
	isDebugMsg   bool
	isConsoleMsg bool
}

func reportConsoleVolumeMessage(msg string, a Volumes, b Volumes) {
	reportVolumeMessage(msg, a, b, false, true)
}

func reportDebugVolumeMessage(msg string, a Volumes, b Volumes) {
	reportVolumeMessage(msg, a, b, true)
}

func reportVolumeMessage(msg string, a Volumes, b Volumes, flags ...bool) {
	log := *(logging.SilentLogSet().Child("Volumes").(*logging.LogSet))
	console := false
	debugStmt := false
	if len(flags) > 0 {
		debugStmt = flags[0]
		if len(flags) > 1 {
			console = flags[1]
		}
	}

	msgLog := volumeMessage{
		msg:          msg,
		CallerInfo:   logging.GetCallerInfo(logging.NotHere()),
		volumePairA:  a,
		volumePairB:  b,
		isDebugMsg:   debugStmt,
		isConsoleMsg: console,
	}
	logging.Deliver(log, msgLog)
}

func (msg volumeMessage) WriteToConsole(console io.Writer) {
	if msg.isConsoleMsg {
		fmt.Fprintf(console, "%s\n", msg.composeMsg())
	}
}

func (msg volumeMessage) DefaultLevel() logging.Level {
	level := logging.WarningLevel
	if msg.isDebugMsg {
		level = logging.DebugLevel
	}

	return level
}

func (msg volumeMessage) Message() string {
	return msg.composeMsg()
}

func (msg volumeMessage) composeMsg() string {
	return fmt.Sprintf("%s:  volume pair A %v, volume pair B %v", msg.msg, msg.volumePairA, msg.volumePairB)
}

func (msg volumeMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousGenericV1)
	msg.CallerInfo.EachField(f)
}
