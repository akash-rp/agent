package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"

	"github.com/labstack/echo/v4"
)

var RootPath string = "/usr/local/lsws/conf/vhosts"

func addNewSite(wp wpadd) error {
	writeFile := fmt.Sprintf("include %s/%s.d/domain/*.conf", RootPath, wp.AppName)
	if err := ioutil.WriteFile(fmt.Sprintf("%s/%s.conf", RootPath, wp.AppName), []byte(writeFile), 0750); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error creating app conf")
	}
	err := addDomainConf(wp.Domain, wp.AppName)
	if err != nil {
		return err
	}
	err = addMainConf(wp)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(RootPath+fmt.Sprintf("/%s.d/modules", wp.AppName), 0750); err != nil {
		return errors.New("error creating domain directory")
	}
	err = addExtPhp(wp)
	if err != nil {
		return err
	}
	us, _ := user.Lookup("lsadm")
	gp, _ := user.LookupGroup("nogroup")
	userID, _ := strconv.Atoi(us.Uid)
	grpId, _ := strconv.Atoi(gp.Gid)
	file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/phpini.conf", wp.AppName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	if err != nil {
		return errors.New("error opening phpini file")
	}
	file.Write([]byte(`
phpIniOverride{
    php_value newrelic.enabled false
}`))
	file.Close()
	if err := os.Chown(fmt.Sprintf("%s/%s.d/modules/extphp.conf", RootPath, wp.AppName), userID, grpId); err != nil {
		return errors.New("extphp permission error")
	}
	if err := os.Chown(fmt.Sprintf("%s/%s.d/modules/phpini.conf", RootPath, wp.AppName), userID, grpId); err != nil {
		return errors.New("phpini permission error")
	}
	if err := os.Chown(fmt.Sprintf("%s/%s.d/main.conf", RootPath, wp.AppName), userID, grpId); err != nil {
		return errors.New("main conf permission error")
	}
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	return nil
}

func addExtPhp(wp wpadd) error {
	path := RootPath + fmt.Sprintf("/%s.d/modules", wp.AppName)
	exphp := fmt.Sprintf(`
	index {
		useServer 0
		indexFiles index.php index.html
	}
	
	scripthandler {
		add lsapi:lsphp_%s php
	}
extprocessor lsphp_%s {
	type lsapi
	address uds://tmp/lshttpd/lsphp-%s.sock
	maxConns 5
	env PHP_LSAPI_MAX_REQUESTS=5000
	env PHP_LSAPI_CHILDREN=5
	env PHPRC=/usr/local/lsws/php-ini/%s/php.ini
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
}`, wp.AppName, wp.AppName, wp.AppName, wp.AppName, wp.UserName, wp.UserName)

	if err := ioutil.WriteFile(fmt.Sprintf("%s/extphp.conf", path), []byte(exphp), 0750); err != nil {
		return errors.New("error writing extphp")
	}
	return nil
}

func addDomainConf(Domain Domain, appName string) error {
	path := RootPath + fmt.Sprintf("/%s.d/domain", appName)

	if err := os.MkdirAll(path, 0750); err != nil {
		return errors.New("error creating domain directory")
	}
	var confUrl string
	if Domain.IsSubDomain {
		confUrl = Domain.Url
	} else {
		confUrl = Domain.Url + ", " + "www." + Domain.Url
	}
	writeFile := fmt.Sprintf(
		`
# Editing this file manually might change litespeed behavior,
# Make sure you know what are you doing
virtualhost %s {
  listeners http, https
	
  vhDomain                  %s	
  
  rewrite  {
 	enable                  1
  	autoLoadHtaccess        1
  }
		
  include %s/%s.d/main.conf
}
`, Domain.Url, confUrl, RootPath, appName)
	if err := ioutil.WriteFile(fmt.Sprintf("%s/%s.conf", path, Domain.Url), []byte(writeFile), 0750); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error creating app conf")
	}

	return nil
}

func addMainConf(wp wpadd) error {
	path := RootPath + fmt.Sprintf("/%s.d", wp.AppName)
	second := fmt.Sprintf(`
	vhRoot /home/%[1]s/%[2]s
	allowSymbolLink 1
	enableScript 1
	restrained 1
	setUIDMode 0

	docRoot $VH_ROOT/public/	
	enableGzip 1
	enableIpGeo 0

	errorlog /var/log/hosting/%[2]s/lsws_error.log {
		useServer 0
		logLevel ERROR
		rollingSize 10M
	}
	
	accesslog var/log/hosting/%[2]s/lsws_access.log {
		useServer 0
		keepDays 10
		compressArchive 1
		rollingSize 10M
	}
	
	expires  {
		enableExpires           1
		expiresByType           image/*=A604800,text/css=A604800,application/x-javascript=A604800,application/javascript=A604800,font/*=A604800,application/x-font-ttf=A604800
	}
	
	include /usr/local/lsws/conf/vhosts/%[2]s.d/modules/*.conf
`, wp.UserName, wp.AppName)

	if err := ioutil.WriteFile(fmt.Sprintf("%s/main.conf", path), []byte(second), 0750); err != nil {
		return errors.New("cannot write main conf")
	}

	return nil

}
