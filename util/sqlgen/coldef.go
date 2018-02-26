// Package sqlgen allows for SQL queries to be automatically generated.
package sqlgen

type coldef struct {
	fmt, name string
	candidate bool
}
