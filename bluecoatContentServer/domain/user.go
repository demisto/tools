package domain

import (
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// User for basic auth
type User struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// GetHashFromPassword returns the hash based on bcrypt
func GetHashFromPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return base64.StdEncoding.EncodeToString(hash)
}

// SetPassword sets the password on the user with bcrypt
func (u *User) SetPassword(password string) {
	u.Password = GetHashFromPassword(password)
}

// ValidPassword ...
func (u *User) ValidPassword(p string) bool {
	data, err := base64.StdEncoding.DecodeString(u.Password)
	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword(data, []byte(p)) == nil
}
