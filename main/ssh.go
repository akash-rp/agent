package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func getSshUsers(c echo.Context) error {
	out, err := exec.Command("/bin/bash", "-c", "w").Output()

	if err != nil {
		return c.JSON(400, err.Error())
	}
	list := strings.Split(string(out), "\n")
	list = list[2:]
	type sshInfo struct {
		User    string `json:"user"`
		Id      string `json:"id"`
		Ip      string `json:"ip"`
		Login   string `json:"login"`
		Ideal   string `json:"ideal"`
		Process string `json:"process"`
	}
	users := []sshInfo{}

	for _, ssh := range list {
		ssh = strings.Join(strings.Fields(ssh), " ")
		result := strings.Split(ssh, " ")
		if len(result) > 1 {
			user := sshInfo{
				User:    result[0],
				Id:      result[1],
				Ip:      result[2],
				Login:   result[3],
				Ideal:   result[4],
				Process: result[7],
			}
			users = append(users, user)
		}
	}
	return c.JSON(200, users)
}

func killSshSession(c echo.Context) error {
	type id struct {
		User string `json:"user"`
	}
	user := new(id)
	c.Bind(&user)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("pkill -9 -t %s", user.User)).Output()
	return c.HTML(200, "")
}
