package models

import (
	"errors"
	"fmt"
)

var (
	ErrNoRecognizedCommand  = errors.New("not valid command")
	ErrServiceNotExistsInDB = errors.New("service not exists in databse")
	ErrMsgNotACommand       = errors.New("msg must be only command, starting with '/'")
	ErrEmptyMsg             = errors.New("empty message")
)

// User represents the user, that can hold many creds
type User struct {
	ID       string    `json:"_id" structs:"_id" mapstructure:"_id"`
	Rev      string    `json:"_rev" structs:"_rev" mapstructure:"_rev,omitempty"`
	Username string    `json:"username" structs:"username" mapstructure:"username"`
	Services []Service `json:"services" structs:"services" mapstructure:"services"`
}

// Service contains creds, including service's name itself
type Service struct {
	Name     string `json:"name" structs:"name" mapstructure:"name"`
	Login    string `json:"login" structs:"login" mapstructure:"login"`
	Password string `json:"password" structs:"password" mapstructure:"password"`
}

func (s Service) String() string {
	return fmt.Sprintf("service: %s\nlogin: %s pass: %s\n", s.Name, s.Login, s.Password)
}
