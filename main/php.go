package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
	"gopkg.in/ini.v1"
)

func changePHP(c echo.Context) error {
	PHPDetails := new(PHPChange)
	c.Bind(&PHPDetails)
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s|path /usr/local/lsws/%s/bin/lsphp|path /usr/local/lsws/%s/bin/lsphp|' /usr/local/lsws/conf/vhosts/%s.d/modules/extphp.conf", PHPDetails.OldPHP, PHPDetails.NewPHP, PHPDetails.Name)).Output()
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return c.String(http.StatusOK, "success")
}

func getPHPini(c echo.Context) error {
	name := c.Param("name")
	path := fmt.Sprintf("/usr/local/lsws/php-ini/%s/php.ini", name)
	cfg, err := ini.Load(path)
	if err != nil {
		return c.JSON(400, "File not found")
	}
	var php PHP
	cfg.Section("PHP").MapTo(&php)
	return c.JSON(http.StatusOK, php)
}

func updatePHPini(c echo.Context) error {
	php := new(PHPini)
	c.Bind(&php)
	name := c.Param("name")
	path := fmt.Sprintf("/usr/local/lsws/php-ini/%s/php.ini", name)

	cfg := ini.Empty()
	ini.ReflectFrom(cfg, php)
	cfg.SaveTo(path)
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	cfg, err := ini.Load(path)
	if err != nil {
		return c.JSON(400, "File not found")
	}
	var phpGet PHP
	cfg.Section("PHP").MapTo(&phpGet)
	return c.JSON(http.StatusOK, phpGet)
}

func getPHPSettings(c echo.Context) error {
	appName := c.Param("name")
	file, err := os.Open(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/extphp.conf", appName))
	if err != nil {
		log.Print(err.Error())
		return c.NoContent(400)
	}
	type config struct {
		MaxConnections int `json:"maxConn"`
	}
	// w := bufio.NewWriter(os.Stdout)
	requiredOptions := []string{"maxConns", "env", "initTimeout", "retryTimeout", "instances"}
	scanner := bufio.NewScanner(file)
	stringArrays := [][]string{}
	jsonString := "{"
	for scanner.Scan() {
		stringArrays = append(stringArrays, strings.Split(strings.TrimSpace(scanner.Text()), " "))
	}

	for _, single := range stringArrays {
		if contains(requiredOptions, single[0]) {
			if single[0] == "env" {

				single = append(single[:0], single[1:]...)
				single = strings.Split(single[0], "=")
				if single[0] == "PHPRC" {
					continue
				}
			}
			jsonString = jsonString + fmt.Sprintf("\"%s\":%s,", single[0], single[1])
		}
	}
	jsonString = strings.TrimSuffix(jsonString, ",")
	jsonString = jsonString + "}"
	var final interface{}
	json.Unmarshal([]byte(jsonString), &final)
	return c.JSON(200, final)
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func updatePHPsettings(c echo.Context) error {
	name := c.Param("name")
	type Settings struct {
		User     string `json:user`
		Settings struct {
			PhpLsapiChildren       int `json:"PHP_LSAPI_CHILDREN"`
			PhpLsapiMaxIdle        int `json:"PHP_LSAPI_MAX_IDLE"`
			PhpLsapiMaxProcessTime int `json:"PHP_LSAPI_MAX_PROCESS_TIME"`
			PhpLsapiMaxRequests    int `json:"PHP_LSAPI_MAX_REQUESTS"`
			PhpLsapiSlowReqMsecs   int `json:"PHP_LSAPI_SLOW_REQ_MSECS"`
			InitTimeout            int `json:"initTimeout"`
			Instances              int `json:"instances"`
			MaxConns               int `json:"maxConns"`
			RetryTimeout           int `json:"retryTimeout"`
		} `json:"settings"`
	}
	data := new(Settings)
	c.Bind(&data)
	settings := data.Settings
	log.Print(settings)
	phpSettings := fmt.Sprintf(`
extprocessor lsphp_%[1]s {
	type lsapi
	address uds://tmp/lshttpd/lsphp-%[1]s.sock
	maxConns %[2]d
	env PHP_LSAPI_MAX_REQUESTS=%[3]d
	env PHP_LSAPI_CHILDREN=%[4]d
	env PHPRC=/usr/local/lsws/php-ini/%[1]s/php.ini
	env PHP_LSAPI_MAX_IDLE=%[5]d
	env PHP_LSAPI_MAX_PROCESS_TIME=%[6]d
	env PHP_LSAPI_SLOW_REQ_MSECS=%[11]d
	initTimeout %[7]d
	retryTimeout %[8]d
	persistConn 1
	respBuffer 0
	autoStart 2
	path /usr/local/lsws/lsphp74/bin/lsphp
	backlog 100
	instances %[9]d
	extUser %[10]s
	extGroup %[10]s
	runOnStartUp 3
	priority 0
	memSoftLimit 2047M
	memHardLimit 2047M
	procSoftLimit 400
	procHardLimit 500
}`, name, settings.MaxConns, settings.PhpLsapiMaxRequests, settings.PhpLsapiChildren, settings.PhpLsapiMaxIdle, settings.PhpLsapiMaxProcessTime, settings.InitTimeout, settings.RetryTimeout, settings.Instances, data.User, settings.PhpLsapiSlowReqMsecs)
	err := ioutil.WriteFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/extphp.conf", name), []byte(phpSettings), 0750)
	if err != nil {
		log.Print(err.Error())
		return c.NoContent(400)
	}
	defer exec.Command("/bin/bash", "-c", "service lsws reload; killall lsphp").Output()
	return getPHPSettings(c)
}
