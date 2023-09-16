package main

import (
	"encoding/json"
	"os"
)

type Auth struct {
	Passwords map[string]string
}

func (auth *Auth) LoadPasswords(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	defer f.Close()
	var passwords map[string]string
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&passwords); err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	auth.Passwords = passwords
	return nil
}

func (auth *Auth) IsPasswordValid(username, password string) bool {
	if auth.Passwords == nil {
		return false
	}
	if p, found := auth.Passwords[username]; found && p == password {
		return true
	}
	return false
}

func NewAuth() *Auth {
	auth := Auth{
		Passwords: make(map[string]string),
	}
	return &auth
}
