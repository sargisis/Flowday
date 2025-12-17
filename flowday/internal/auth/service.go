package auth 

import (

	"errors"

	"flowday/internal/db"
	"flowday/internal/models"
)

func Register(email , password string) (*models.User , error) {
	hashed , err := HashPassword(password);

	if err != nil {
		return nil , err 
	}

	user := models.User {
		Email: email,
		Password: hashed,
	}

	if err := db.DB.Create(&user).Error; err != nil {
		return nil , err

	}

	return &user , nil
}

func Login(email, password string) (string, error) {
	var user models.User

	if err := db.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return "", errors.New("invalid credentials")
	}

	if !CheckPassword(user.Password, password) {
		return "", errors.New("invalid credentials")
	}

	return GenerateToken(user.ID)
}