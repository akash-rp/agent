package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func getBannedIpList(c echo.Context) error {
	ips, err := exec.Command("/bin/bash", "-c", "fail2ban-client get sshd banned").Output()
	log.Print(ips)
	if err != nil {
		return c.JSON(400, "something went wrong")
	}
	str := strings.ReplaceAll(string(ips), "'", "\"")

	var obj []string
	err = json.Unmarshal([]byte(str), &obj)
	if err != nil {
		log.Print(err.Error())
		log.Print("Unmarshal error")
		return c.JSON(400, "something went wrong")
	}
	return c.JSON(200, obj)
}
