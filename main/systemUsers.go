package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func getSystemUsers(c echo.Context) error {
	out, err := exec.Command("/bin/bash", "-c", "awk -F: '($3>=1000)&&($1!=\"nobody\"){print $1}' /etc/passwd").Output()
	if err != nil {
		return c.NoContent(400)
	}
	log.Print(string(out))
	trimmedString := strings.TrimRight(string(out), "\n")
	list := strings.Split(string(trimmedString), "\n")
	type User struct {
		User          string `json:"user"`
		NumberOfSites int    `json:"sites"`
	}
	type Users []User
	users := new(Users)
	*users = append(*users, User{User: "root", NumberOfSites: 0})
	for _, user := range list {
		if user == "" {
			continue
		}
		numSite := 0
		for _, site := range obj.Sites {
			if user == site.User {
				numSite++
			}
		}
		*users = append(*users, User{User: user, NumberOfSites: numSite})
	}
	return c.JSON(200, users)

}

func changeUserPassword(c echo.Context) error {
	type User struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	user := new(User)
	c.Bind(&user)
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("echo \"%s:%s\" | chpasswd", user.User, user.Password)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		return c.NoContent(400)
	}
	return c.NoContent(200)
}

func deleteSystemUser(c echo.Context) error {
	type User struct {
		User string `json:"user"`
	}
	user := new(User)
	c.Bind(&user)
	if user.User == "root" {
		return c.NoContent(400)
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("deluser -remove-home %s", user.User)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		return c.NoContent(400)
	}
	return getSystemUsers(c)
}
