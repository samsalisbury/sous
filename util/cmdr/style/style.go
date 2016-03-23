package style

import (
	"strconv"
	"strings"
)

type (
	Code  int
	Style []Code
)

var (
	defaultStyle = Style{NoAttribute, DefaultColor, DefaultBGColor}
)

func DefaultStyle() Style { return defaultStyle }

const (

	// Attributes
	NoAttribute Code = 0
	Bold             = 1
	Dim              = 2
	Italic           = 3
	Underline        = 4
	Blink            = 5
	FastBlink        = 6
	Invert           = 7
	Hidden           = 8

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

func (s *Style) Add(codes ...Code) {
	(*s) = append(*s, codes...)
}

func (s *Style) Remove(codes ...Code) {
	other := Style(codes)
	(*s) = s.Subtract(other)
}

func (s Style) Subtract(other Style) Style {
	out := Style{}
	for _, c := range s {
		if !other.Contains(c) {
			out = append(out, c)
		}
	}
	return out
}

func (s Style) Contains(c Code) bool {
	for _, d := range s {
		if d == c {
			return true
		}
	}
	return false
}

func (s Style) String() string {
	strs := make([]string, len(s))
	for i, n := range s {
		strs[i] = strconv.Itoa(int(n))
	}
	return strings.Join(strs, ";")
}
