package sous

import "fmt"

// User represents a user of the Sous client.
type User struct {
	// Name is the full name of this user.
	Name string `env:"SOUS_USER_NAME"`
	// Email is the email address of this user.
	Email string `env:"SOUS_USER_EMAIL"`
}

// String returns the name and email in standard email address format, i.e.:
//
//     User Name <email@address.com>
//
// This is used directly by git commit --author, and is reasonable in logs etc.
func (u User) String() string {
	return fmt.Sprintf("%s <%s>", u.Name, u.Email)
}
