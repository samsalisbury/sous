package cmdr

import (
	"strconv"
	"strings"
)

type (
	EscapeCode int
	Style      []EscapeCode
)

var (
	defaultStyle = Style{NoAttribute, DefaultColor, DefaultBGColor}
)

const (

	// Attributes
	NoAttribute EscapeCode = 0
	Bold                   = 1
	Dim                    = 2
	Italic                 = 3
	Underline              = 4
	Blink                  = 5
	FastBlink              = 6
	Invert                 = 7
	Hidden                 = 8

	// Foreground Colours
	Black        = 30
	Red          = 31
	Green        = 32
	Yellow       = 33
	Blue         = 34
	Magenta      = 35
	Cyan         = 36
	White        = 37
	DefaultColor = 39

	// Background Colours
	BGBlack        = 40
	BGRed          = 41
	BGGreen        = 42
	BGYellow       = 43
	BGBlue         = 44
	BGMagenta      = 45
	BGCyan         = 46
	BGWhite        = 47
	DefaultBGColor = 49
)

func (s *Style) Add(c ...EscapeCode) {
	(*s) = append(*s, c...)
}

func (s Style) String() string {
	strs := make([]string, len(s))
	for i, n := range s {
		strs[i] = strconv.Itoa(int(n))
	}
	return strings.Join(strs, ";")
}
