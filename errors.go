package main

type (
	UserError struct {
		Message, Tip string
	}
)

func (err UserError) Error() string {
	return err.Message
}
