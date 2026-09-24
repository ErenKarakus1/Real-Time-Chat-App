package auth

import "golang.org/x/crypto/bcrypt"

const passwordCost = bcrypt.DefaultCost

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), passwordCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func CheckPassword(password string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
