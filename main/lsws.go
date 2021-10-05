package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"

	"github.com/labstack/echo/v4"
)

func editLsws(wp wpadd) error {
	first := fmt.Sprintf(
		`
virtualhost %s {
    vhRoot /home/%s/%s/
    listeners Default
    configFile $SERVER_ROOT/conf/vhosts/%s.d/main.conf
    allowSymbolLink 1
    enableScript 1
    restrained 1
    setUIDMode 0
}`,
		wp.AppName, wp.UserName, wp.AppName, wp.AppName)
	if err := os.Chdir("/usr/local/lsws/conf/vhosts"); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Changing directory error 1")
	}
	if err := ioutil.WriteFile(fmt.Sprintf("%s.conf", wp.AppName), []byte(first), 0750); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 2")
	}
	us, _ := user.Lookup("lsadm")
	gp, _ := user.LookupGroup("nogroup")
	userID, _ := strconv.Atoi(us.Uid)
	grpId, _ := strconv.Atoi(gp.Gid)
	if err := os.Chown(fmt.Sprintf("%s.conf", wp.AppName), userID, grpId); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 3")
	}
	if err := os.Mkdir(fmt.Sprintf("%s.d", wp.AppName), os.FileMode(0750)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 4")
	}
	if err := os.Chown(fmt.Sprintf("%s.d", wp.AppName), userID, grpId); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 5")
	}
	second := fmt.Sprintf(`
docRoot $VH_ROOT/
VhDomain %s
enableGzip 1

errorlog /var/log/hosting/%s/lsws_error.log {
useServer 0
logLevel ERROR
rollingSize 10M
}

accesslog var/log/hosting/%s/lsws_access.log {
useServer 0
rollingSize 10M
keepDays 10
compressArchive 1
}

index {
useServer 0
indexFiles index.php index.html
}

scripthandler {
add lsapi:lsphp_%s php
}

include /usr/local/lsws/conf/vhosts/%s.d/handlers/*.conf

expires  {
  enableExpires           1
  expiresByType           image/*=A604800,text/css=A604800,application/x-javascript=A604800,application/javascript=A604800,font/*=A604800,application/x-font-ttf=A604800
}

rewrite {
enable 1
autoLoadHtaccess 1
}`, wp.Url, wp.AppName, wp.AppName, wp.AppName, wp.AppName)
	if err := os.Chdir(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d", wp.AppName)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 6")
	}
	if err := ioutil.WriteFile("main.conf", []byte(second), 0750); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 7")
	}
	if err := os.Chown("main.conf", userID, grpId); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 8")
	}
	if err := os.MkdirAll("handlers", 0750); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 9")
	}
	if err := os.Chown("handlers", userID, grpId); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 10")
	}
	if err := os.Chdir(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/handlers", wp.AppName)); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 11")
	}
	third := fmt.Sprintf(`
extprocessor lsphp_%s {
type lsapi
address uds://tmp/lshttpd/lsphp-%s.sock
maxConns 35
env PHP_LSAPI_MAX_REQUESTS=5000
env PHP_LSAPI_CHILDREN=35
env PHPRC=/usr/local/lsws/php/%s/php.ini
initTimeout 60
retryTimeout 0
persistConn 1
respBuffer 0
autoStart 2
path /usr/local/lsws/lsphp74/bin/lsphp
backlog 100
instances 1
extUser %s
extGroup %s
runOnStartUp 3
priority 0
memSoftLimit 2047M
memHardLimit 2047M
procSoftLimit 400
procHardLimit 500
}`, wp.AppName, wp.AppName, wp.AppName, wp.UserName, wp.UserName)

	if err := ioutil.WriteFile("extphp.conf", []byte(third), 0750); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 12")
	}
	if err := os.Chown("extphp.conf", userID, grpId); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error 13")
	}
	os.Chdir("/usr/Hosting/")
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	return nil
}
