package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func enableGeoip(c echo.Context) error {
	app := c.Param("site")
	out, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/enableIpGeo/c\\enableIpGeo        1' /usr/local/lsws/conf/vhosts/%s.d/main.conf", app)).CombinedOutput()
	log.Print(string(out))
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	return c.JSON(200, "success")
}

func disableGeoip(c echo.Context) error {
	app := c.Param("site")
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/enableIpGeo/c\\enableIpGeo        0' /usr/local/lsws/conf/vhosts/%s.d/main.conf", app)).Output()
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	return c.JSON(200, "success")
}
