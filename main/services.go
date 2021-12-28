package main

import (
	"os/exec"

	"github.com/labstack/echo/v4"
)

func getServiceStatus(c echo.Context) error {
	exec.Command("/bin/bash", "-c", "systemctl check hosting lsws mariadb")
	return nil
}
