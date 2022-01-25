package main

import (
	"os/exec"

	"github.com/labstack/echo/v4"
)

func getServiceStatus(c echo.Context) error {
	exec.Command("/bin/bash", "-c", "systemctl check lsws mariadb")
	return nil
}
