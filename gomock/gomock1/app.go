package app

import (
	"database/sql"
)

type IUserRepository interface {
	InsertAUser(user *UserModel) error
}

type UserModel struct {
	ID    uint64         `db:"id"`
	Name  sql.NullString `db:"name"`
	Email string         `db:"email"`
}

type User struct {
	UserID uint32 `json:"user_id"`
	Name   string `json:"user_name"`
	Email  string `json:"user_email"`
}

type userService struct {
	userRepo IUserRepository
}

func (s *userService) RegisterUser(user *User) error {
	var userModel UserModel
	// Mapping process part
	// ID
	userModel.ID = uint64(user.UserID)
	// Name
	if user.Name == "" {
		userModel.Name = sql.NullString{}
	} else {
		userModel.Name = sql.NullString{
			String: user.Name,
			Valid:  true,
		}
	}
	// Email
	userModel.Email = user.Email

	// Call repository func
	err := s.userRepo.InsertAUser(&userModel) // この&userModelの内容のチェックをテストでしたい。
	if err != nil {
		return err
	}

	return nil
}
