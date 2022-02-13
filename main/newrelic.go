package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func enabelNewrelic(c echo.Context) error {
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "service newrelic-daemon start").Output()
	exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.JSON(200, "success")
}

func disableNewrelic(c echo.Context) error {
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic.ini /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic.ini /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic.ini /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic.ini /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "service newrelic-daemon stop").Output()
	exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.JSON(200, "success")
}

func enabelNewrelicPerSite(c echo.Context) error {
	conf := new(struct {
		App string `json:"app"`
		Key string `json:"key"`
	})
	c.Bind(&conf)

	file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/phpini.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	if err != nil {
		return c.JSON(400, "Error opening file")
	}
	file.Write([]byte(fmt.Sprintf(`
phpIniOverride{
    php_value newrelic.appname "%s"
    php_value newrelic.license "%s"
}
`, conf.App, conf.Key)))
	file.Close()
	go exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	go exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.JSON(200, "success")
}

func disableNewrelicPerSite(c echo.Context) error {
	conf := new(struct {
		App string `json:"app"`
	})
	c.Bind(&conf)

	file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/phpini.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	if err != nil {
		return c.JSON(400, "Error opening file")
	}
	file.Write([]byte(`
phpIniOverride{
    php_value newrelic.enabled false
}`))
	file.Close()
	go exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	go exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.JSON(200, "success")
}
