package helpers

import (
	"errors"
)

type UserType struct {
	ID         uint32
	TelegramID uint32
	UserName   string
	FirstName  string
	LastName   string
	IsAdmin    bool
	Nodes      *[]*NodeType
	LastMenuID int
}

var Users []*UserType

func GetUserByTelegramID(tgID uint32) (*UserType, error) {
	for _, u := range Users {
		if tgID == u.TelegramID {
			return u, nil
		}
	}
	return nil, errors.New("User not found")
}
