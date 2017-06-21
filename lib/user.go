package sous

import "strings"

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
	parts := []string{}
	if u.Name != "" {
		parts = append(parts, u.Name)
	}
	if u.Email != "" {
		parts = append(parts, "<"+u.Email+">")
	}

	return strings.Join(parts, " ")
}

// Complete returns true only if both Name and Email have a non-empty value
func (u User) Complete() bool {
	return u.Name != "" && u.Email != ""
}

// HTTPHeaders returns a map suitable to use as HTTP headers to be consumed by the server.
func (u User) HTTPHeaders() map[string]string {
	return map[string]string{
		"Sous-User-Name":  u.Name,
		"Sous-User-Email": u.Email,
	}
}
