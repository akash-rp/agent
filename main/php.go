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
	"reflect"
	"strconv"
	"strings"
	"unicode"

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
	cfg.ValueMapper = parseNumberTillNonInteger
	if err != nil {
		return AbortWithErrorMessage(c, "File not found")
	}
	var php PHP
	cfg.Section("PHP").MapTo(&php)
	result := convertStringToIntStruct(php)
	return c.JSON(http.StatusOK, result)
}

func updatePHPini(c echo.Context) error {
	phpParsed := new(PhpIniParsed)
	c.Bind(&phpParsed)
	name := c.Param("name")
	path := fmt.Sprintf("/usr/local/lsws/php-ini/%s/php.ini", name)
	php := ConvertPhpIniParsedToPHP(phpParsed)
	PHPini := &PHPini{php}
	cfg := ini.Empty()
	err := ini.ReflectFrom(cfg, PHPini)
	if err != nil {
		log.Print(err.Error())
		return c.NoContent(400)
	}
	cfg.SaveTo(path)
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	return getPHPini(c)
}

func getPHPSettings(c echo.Context) error {
	appName := c.Param("name")
	file, err := os.Open(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/extphp.conf", appName))
	if err != nil {
		log.Print(err.Error())
		return c.NoContent(400)
	}
	// type config struct {
	// 	MaxConnections int `json:"maxConn"`
	// }
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
		User     string `json:"user"`
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

func parseNumberTillNonInteger(input string) string {
	runes := []rune(input)
	var result []rune

	for _, r := range runes {
		if unicode.IsDigit(r) {
			result = append(result, r)
		} else if len(result) == 0 {
			// If the first character is not a digit, return the original input string
			return input
		} else {
			// Stop parsing when a non-integer character is encountered
			break
		}
	}

	return string(result)
}

func convertStringToIntStruct(source interface{}) map[string]interface{} {
	values := reflect.ValueOf(source)
	types := values.Type()

	result := map[string]interface{}{}

	for i := 0; i < values.NumField(); i++ {
		if values.Field(i).Type().Name() == "string" {
			if s, err := strconv.ParseInt(values.Field(i).String(), 10, 32); err == nil {
				result[types.Field(i).Name] = s
				continue
			}
		}
		result[types.Field(i).Name] = values.Field(i).Interface()
	}

	return result
}

// ConvertPhpIniParsedToPHP converts a PhpIniParsed object to a PHP object
func ConvertPhpIniParsedToPHP(parsed *PhpIniParsed) PHP {
	return PHP{
		MaxExecutionTime:      fmt.Sprintf("%d", parsed.MaxExecutionTime),
		MaxFileUploads:        fmt.Sprintf("%d", parsed.MaxFileUploads),
		MaxInputTime:          fmt.Sprintf("%d", parsed.MaxInputTime),
		MaxInputVars:          fmt.Sprintf("%d", parsed.MaxInputVars),
		MemoryLimit:           fmt.Sprintf("%dM", parsed.MemoryLimit),
		PostMaxSize:           fmt.Sprintf("%dM", parsed.PostMaxSize),
		SessionCookieLifetime: fmt.Sprintf("%d", parsed.SessionCookieLifetime),
		SessionGcMaxlifetime:  fmt.Sprintf("%d", parsed.SessionGcMaxlifetime),
		ShortOpenTag:          parsed.ShortOpenTag,
		UploadMaxFilesize:     fmt.Sprintf("%dM", parsed.UploadMaxFilesize),
		Timezone:              parsed.Timezone,
		OpenBaseDir:           parsed.OpenBaseDir,
	}
}
