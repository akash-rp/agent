package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func getBannedIpList(c echo.Context) error {
	ips, err := exec.Command("/bin/bash", "-c", "fail2ban-client get sshd banned").Output()
	log.Print(ips)
	if err != nil {
		return AbortWithErrorMessage(c, "something went wrong")
	}
	str := strings.ReplaceAll(string(ips), "'", "\"")

	var obj []string
	err = json.Unmarshal([]byte(str), &obj)
	if err != nil {
		log.Print(err.Error())
		log.Print("Unmarshal error")
		return AbortWithErrorMessage(c, "something went wrong")
	}
	return c.JSON(200, obj)
}

func unbanIp(c echo.Context) error {
	type IP struct {
		Ip string `json:"ip"`
	}
	ip := new(IP)
	c.Bind(&ip)
	_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("fail2ban-client unban %s", ip.Ip)).Output()
	if err != nil {
		log.Print(err.Error())
		return c.NoContent(400)
	}
	return getBannedIpList(c)
}
