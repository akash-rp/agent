package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/labstack/echo/v4"
)

func enableNewrelicRequest(c echo.Context) error {
	conf := new(struct {
		Duration int `json:"duration"`
	})
	c.Bind(&conf)
	enableNewrelic(conf.Duration)
	return c.JSON(200, "success")
}

func enableNewrelic(duration int) {
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic.ini").Output()
	exec.Command("/bin/bash", "-c", "service newrelic-daemon start").Output()
	exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	if duration > 0 {

		cronInt.Every(1).Day().StartAt(time.Now().Add(time.Hour * time.Duration(duration))).LimitRunsTo(1).Tag("Newrelic").Do(func() {
			disableNewrelic()
		})
	}
}

func disableNewrelicRequest(c echo.Context) error {
	disableNewrelic()
	return c.JSON(200, "success")
}

func disableNewrelic() {
	cronInt.RemoveByTag("Newrelic")
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic.ini /usr/local/lsws/lsphp74/etc/php/7.4/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic.ini /usr/local/lsws/lsphp73/etc/php/7.3/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic.ini /usr/local/lsws/lsphp72/etc/php/7.2/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "mv /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic.ini /usr/local/lsws/lsphp80/etc/php/8.0/mods-available/newrelic").Output()
	exec.Command("/bin/bash", "-c", "service newrelic-daemon stop").Output()
	exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	exec.Command("/bin/bash", "-c", "killall lsphp").Output()
}

func enabelNewrelicPerSite(c echo.Context) error {
	conf := new(struct {
		App string `json:"app"`
		Key string `json:"key"`
	})
	c.Bind(&conf)

	file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/phpini.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	if err != nil {
		return AbortWithErrorMessage(c, "Error opening file")
	}
	file.Write([]byte(fmt.Sprintf(`
phpIniOverride{
    php_value newrelic.appname "%s"
    php_value newrelic.license "%s"
}
`, conf.App, conf.Key)))
	file.Close()
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.JSON(200, "success")
}

func disableNewrelicPerSite(c echo.Context) error {
	conf := new(struct {
		App string `json:"app"`
	})
	c.Bind(&conf)

	file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/phpini.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	if err != nil {
		return AbortWithErrorMessage(c, "Error opening file")
	}
	file.Write([]byte(`
phpIniOverride{
    php_value newrelic.enabled false
}`))
	file.Close()
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.JSON(200, "success")
}
