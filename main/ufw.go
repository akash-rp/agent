package main

import (
	"encoding/json"
	"log"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func getUfwRules(c echo.Context) error {
	out, err := exec.Command("/bin/bash", "-c", "ufw status numbered | jc --ufw").CombinedOutput()
	type Ufw struct {
		Status string
		Rules  []struct {
			Index            int
			Action           string
			Network_protocol string
			To_ports         []int
			To_transport     string
			From_ip          string
		}
	}
	if err != nil {
		log.Print("First")
		log.Print(string(out))
		return c.JSON(400, err.Error())
	}
	log.Print(string(out))
	ufw := new(Ufw)
	err = json.Unmarshal(out, &ufw)
	if err != nil {
		log.Print("Second")

		return c.JSON(400, err.Error())
	}
	return c.JSON(200, ufw)
}
