package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	"github.com/labstack/echo/v4"
	"gopkg.in/ini.v1"
)

func changePHP(c echo.Context) error {
	PHPDetails := new(PHPChange)
	c.Bind(&PHPDetails)
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s|path /usr/local/lsws/%s/bin/lsphp|path /usr/local/lsws/%s/bin/lsphp|' /usr/local/lsws/conf/vhosts/%s.d/handlers/extphp.conf", PHPDetails.OldPHP, PHPDetails.NewPHP, PHPDetails.Name)).Output()
	go exec.Command("/bin/bash", "-c", "service lshttpd restart").Output()
	go exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.String(http.StatusOK, "success")

}

func getPHPini(c echo.Context) error {
	name := c.Param("name")
	path := fmt.Sprintf("/usr/local/lsws/php-ini/%s/php.ini", name)
	cfg, _ := ini.Load(path)
	var php PHP
	cfg.Section("PHP").MapTo(&php)
	return c.JSON(http.StatusOK, php)
}

func updatePHPini(c echo.Context) error {
	php := new(PHPini)
	c.Bind(&php)
	name := c.Param("name")
	cfg := ini.Empty()
	ini.ReflectFrom(cfg, php)
	cfg.SaveTo(fmt.Sprintf("/usr/local/lsws/php-ini/%s/php.ini", name))
	go exec.Command("/bin/bash", "-c", "service lshttpd restart").Output()
	go exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.JSON(http.StatusOK, "success")
}
