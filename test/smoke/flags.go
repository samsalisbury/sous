package smoke

// Flags represents a set of flags to be passed to a Binary.
type Flags interface {
	// FlagMap returns a map of flag name -> flag value.
	FlagMap() map[string]string
	// FlagPrefix returns the standard flag prefix for this set of flags.
	// Typically this will be either "-" or "--".
	FlagPrefix() string
}
