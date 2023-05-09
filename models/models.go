package models

import (
	"errors"
	"fmt"
)

var (
	ErrNoRecognizedCommand = errors.New("not valid command")
	ErrBadParams           = errors.New("bad parameters")
	ErrMsgNotACommand      = errors.New("msg must be only command, starting with '/'")
	ErrEmptyMsg            = errors.New("empty message")
	ErrDatabaseQuery       = errors.New("database error")
)

type User struct {
	Username string    `json:"username"`
	Services []Service `json:"services"`
}

type Service struct {
	Name     string `json:"name"`
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s Service) String() string {
	return fmt.Sprintf("service: %s\nlogin: %s pass: %s\n", s.Name, s.Login, s.Password)
}
