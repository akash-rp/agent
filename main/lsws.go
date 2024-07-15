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
	us, err := user.Lookup("nobody")
	if err != nil {
		return err
	}
	gp, err := user.LookupGroup("nogroup")
	if err != nil {
		return err
	}
	userID, err := strconv.Atoi(us.Uid)
	if err != nil {
		return err
	}
	grpId, err := strconv.Atoi(gp.Gid)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/phpini.conf", wp.AppName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	if err != nil {
		return errors.New("error opening phpini file")
	}
	file.Write([]byte(`
phpIniOverride{
    php_value newrelic.enabled false
}`))
	file.Close()

	err = os.WriteFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/context.conf", wp.AppName), []byte(fmt.Sprintf(`
	context / {
		location              $DOC_ROOT
		allowBrowse             1		
		addDefaultCharset       off		
		accessControl  {
		  allow                 *
		}
		extraHeaders <<<END_extraHeaders
			set Referrer-Policy strict-origin-when-cross-origin
			set Strict-Transport-Security: max-age=31536000
			set X-Content-Type-Options nosniff
			set X-Frame-Options SAMEORIGIN
			set X-XSS-Protection 1; mode=block
		END_extraHeaders
		
		include /usr/local/lsws/conf/vhosts/%s.d/modules/rewrite.conf
	}
	`, wp.AppName)), 0640)
	if err != nil {
		return errors.New("error writing context file")
	}
	//if err := os.WriteFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/rewrite.conf", wp.AppName), []byte(`
	//rewrite {
	//	enable 			 1
	//	autoLoadHtaccess 1
	//}
	//`), 0640); err != nil {
	//	return errors.New("error writing rewrite conf")
	//}
	if err := os.Chown(fmt.Sprintf("%s/%s.d/modules/extphp.conf", RootPath, wp.AppName), userID, grpId); err != nil {
		return errors.New("extphp permission error")
	}
	if err := os.Chown(fmt.Sprintf("%s/%s.d/modules/phpini.conf", RootPath, wp.AppName), userID, grpId); err != nil {
		return errors.New("phpini permission error")
	}
	if err := os.Chown(fmt.Sprintf("%s/%s.d/main.conf", RootPath, wp.AppName), userID, grpId); err != nil {
		return errors.New("main conf permission error")
	}
	//if err := os.Chown(fmt.Sprintf("%s/%s.d/modules/rewrite.conf", RootPath, wp.AppName), userID, grpId); err != nil {
	//	return errors.New("rewrite conf permission error")
	//}
	if err := os.Chown(fmt.Sprintf("%s/%s.d/modules/context.conf", RootPath, wp.AppName), userID, grpId); err != nil {
		return errors.New("context conf permission error")
	}
	defer exec.Command("/bin/bash", "-c", fmt.Sprintf("chown -R nobody:nogroup %s/%s.*", RootPath, wp.AppName)).Output()
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	return nil
}

func addExtPhp(wp wpadd) error {
	path := RootPath + fmt.Sprintf("/%s.d/modules", wp.AppName)
	exphp := fmt.Sprintf(`
extprocessor lsphp_%[1]s {
	type lsapi
	address uds://tmp/lshttpd/lsphp-%[1]s.sock
	maxConns 5
	env PHP_LSAPI_MAX_REQUESTS=5000
	env PHP_LSAPI_CHILDREN=5
	env PHPRC=/usr/local/lsws/php-ini/%[1]s/php.ini
	env PHP_LSAPI_MAX_IDLE=300
	env PHP_LSAPI_MAX_PROCESS_TIME=3600
	env PHP_LSAPI_SLOW_REQ_MSECS=0
	initTimeout 60
	retryTimeout 0
	persistConn 1
	respBuffer 0
	autoStart 2
	path /usr/local/lsws/lsphp74/bin/lsphp
	backlog 100
	instances 1
	extUser %[2]s
	extGroup %[2]s
	runOnStartUp 3
	priority 0
	memSoftLimit 2047M
	memHardLimit 2047M
	procSoftLimit 400
	procHardLimit 500
}`, wp.AppName, wp.UserName)

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
	if Domain.SubDomain {
		confUrl = Domain.Url
	} else {
		confUrl = Domain.Url + ", " + "www." + Domain.Url
	}
	writeFile := fmt.Sprintf(
		`
# Editing this file manually might change litespeed behavior,
# Make sure you know what are you doing
virtualhost %[1]s {
  listeners http, https
	
  vhDomain                  %[2]s	
  
  include %[3]s/%[4]s.d/domain/%[1]s.ssl
  include %[3]s/%[4]s.d/domain/%[1]s.rewrite
  include %[3]s/%[4]s.d/main.conf
}
`, Domain.Url, confUrl, RootPath, appName)
	if err := ioutil.WriteFile(fmt.Sprintf("%s/%s.conf", path, Domain.Url), []byte(writeFile), 0750); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Error creating app conf")
	}
	if !checkIfSslExistsForDomain(Domain) {

		writeFile =
			`
		# Editing this file manually might change litespeed behavior,
		# Make sure you know what are you doing
		
		vhssl{
			
		}
		`
		if err := ioutil.WriteFile(fmt.Sprintf("%s/%s.ssl", path, Domain.Url), []byte(writeFile), 0750); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Error creating app conf")
		}
	} else {
		configureDomainForSSl(appName, Domain.Url)
	}

	writeFile =
		`
# Editing this file manually might change litespeed behavior,
# Make sure you know what are you doing

rewrite  {
	enable                  1
	autoLoadHtaccess        1
 }
`
	if err := ioutil.WriteFile(fmt.Sprintf("%s/%s.rewrite", path, Domain.Url), []byte(writeFile), 0750); err != nil {
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
	
	accesslog /var/log/hosting/%[2]s/lsws_access.log {
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
	index {
		useServer 0
		indexFiles index.php index.html
	}
	
	scripthandler {
		add lsapi:lsphp_%[2]s php
	}

	module cache {
		internal                1
		checkPrivateCache       1
		checkPublicCache        1
		maxCacheObjSize         10000000
		maxStaleAge             200
		qsCache                 1
		reqCookieCache          1
		respCookieCache         1
		ignoreReqCacheCtrl      1
		ignoreRespCacheCtrl     0
	  
		enableCache             0
		expireInSeconds         3600
		enablePrivateCache      0
		privateExpireInSeconds  3600
		ls_enabled              1
		storagePath 			$VH_ROOT/lscache
	  }
`, wp.UserName, wp.AppName)

	if err := ioutil.WriteFile(fmt.Sprintf("%s/main.conf", path), []byte(second), 0750); err != nil {
		return errors.New("cannot write main conf")
	}

	return nil

}
