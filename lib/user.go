package sous

import "fmt"

// User represents a user of the Sous client.
type User struct {
	// Name is the full name of this user.
	Name,
	// Email is the email address of this user.
	Email string
}

func (u User) String() string {
	return fmt.Sprintf("%s <%s>", u.Name, u.Email)
}
