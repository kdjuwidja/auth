package db

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string
	Username string
	Password string
}

func ValidateUser(db *sql.DB, username, password string) (*User, error) {
	var user User

	err := db.QueryRow("SELECT user_id, username, password FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	// Compare the provided password with the hashed password in the database
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	fmt.Printf("ValidateUser %v successfully\n", user)

	return &user, nil
}
