package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nakabonne/tstorage"
)

var obj Config
var cronInt = gocron.NewScheduler(time.UTC)
var metrics, _ = tstorage.NewStorage(
	tstorage.WithTimestampPrecision(tstorage.Milliseconds),
	tstorage.WithDataPath("/usr/Hosting/metrics"),
	tstorage.WithWALBufferedSize(0),
)

// var jobMap = make(map[string]*gocron.Job)

func main() {
	e := echo.New()
	e.Debug = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	data, err := ioutil.ReadFile("/usr/Hosting/config.json")
	if err != nil {
		log.Fatal("Cannot read file")
	}
	json.Unmarshal(data, &obj)
	initCron()
	e.GET("/serverstats", serverStats)
	e.POST("/wp/add", wpAdd)
	e.POST("/wp/delete", wpDelete)
	e.POST("/cert/add", addSslCert)
	e.POST("/cert/reissue", reissueSslCert)
	e.GET("/sites", getSites)
	e.POST("/domain/add", addDomain)
	e.POST("/domain/delete", deleteDomain)
	e.POST("/domain/wildcard/add", addWildcard)
	e.POST("/domain/wildcard/remove", removeWildcard)
	e.POST("/changeprimary", changePrimary)
	e.POST("/changePHP", changePHP)
	e.GET("/getPHPini/:name", getPHPini)
	e.POST("/updatePHPini/:name", updatePHPini)
	e.POST("/updatelocalbackup/:type/:name/:user", updateLocalBackup)
	e.GET("/takelocalondemandbackup/:name/:user", ondemadBackup)
	e.GET("/localbackup/nextrun", nextrun)
	e.GET("/localbackup/list/:name/:user", getLocalBackupsList)
	e.GET("/restorelocalbackup/:name/:user/:mode/:id/:type", restoreBackupFromPanel)
	e.GET("/createstaging/:name/:user/:url/:livesiteurl", createStaging)
	e.GET("/getdbtables/:name/:user", getDatabaseTables)
	e.POST("/syncChanges", syncChanges)
	e.GET("/deleteStaging/:name/:user", deleteStagingSite)
	e.POST("/deleteSite", wpDelete)
	e.POST("/addSSH/:user", addSSHkey)
	e.POST("/removeSSH/:user", removeSSHkey)
	e.GET("/ptlist/:user/:name", getPluginAndThemesStatus)
	e.POST("/ptoperation/:user/:name", updatePluginsThemes)
	// e.POST("/enforceHttps", enforceHttps)
	e.POST("/update7G", update7G)
	e.POST("/updateModsecurity", updateModsecurity)
	e.POST("/newrelic/enable", enabelNewrelicRequest)
	e.POST("/newrelic/enableSite", enabelNewrelicPerSite)
	e.POST("/newrelic/disable", disableNewrelicRequest)
	e.POST("/newrelic/disableSite", disableNewrelicPerSite)
	e.GET("/service/status", getServiceStatus)
	e.POST("/service/restart/:service", serviceRestart)
	e.POST("/service/stop/:service", serviceStop)
	e.POST("/service/start/:service", serviceStart)
	e.POST("/geoip/enable/:site", enableGeoip)
	e.POST("/geoip/disable/:site", disableGeoip)
	e.POST("/ipdeny", updateipdeny)
	e.GET("/metrics", getAllMetrics)
	e.GET("/metrics/:metric/:period", getMetrice)
	e.GET("/ssh/users", getSshUsers)
	e.POST("/ssh/kill", killSshSession)
	e.GET("/ufw/rules", getUfwRules)
	e.GET("/fail2ban/ip", getBannedIpList)
	e.Logger.Fatal(e.Start(":8081"))
}

func serverStats(c echo.Context) error {
	totalmem, _ := exec.Command("/bin/bash", "-c", "free -m | awk 'NR==2{printf $2}'").Output()
	usedmem, _ := exec.Command("/bin/bash", "-c", "free -m | awk 'NR==2{printf $3}'").Output()
	cores, _ := exec.Command("/bin/bash", "-c", "nproc").Output()
	cpuname, _ := exec.Command("/bin/bash", "-c", "lscpu | grep 'Model name' | cut -f 2 -d : | awk '{$1=$1}1'").Output()
	totaldisk, _ := exec.Command("/bin/bash", "-c", " df -h --total -x tmpfs -x devtmpfs -x udev | awk '/total/{printf $2}'").Output()
	useddisk, _ := exec.Command("/bin/bash", "-c", " df -h --total -x tmpfs -x devtmpfs -x udev| awk '/total/{printf $3}'").Output()
	bandwidth, _ := exec.Command("/bin/bash", "-c", "vnstat | awk 'NR==4{print $5$6}'").Output()
	os, err := exec.Command("/bin/bash", "-c", "hostnamectl | grep 'Operating System' | cut -f 2 -d : | awk '{$1=$1}1'").Output()
	uptime, _ := exec.Command("/bin/bash", "-c", "awk '{print int($1/3600)\"h\"\" \"int(($1%3600)/60)\"m\"\" \"int($1%60)\"s\"}' /proc/uptime").Output()
	loadavg, _ := exec.Command("/bin/bash", "-c", "uptime | awk '{ printf \"%s %s %s\",$10,$11,$12}'").Output()
	cpuideal, _ := exec.Command("/bin/bash", "-c", "vmstat | awk 'FNR == 3 {print $15}'").Output()
	if err != nil {
		log.Fatal(err)
	}
	stringBandwith := string(bandwidth)
	stringBandwith = strings.ReplaceAll(stringBandwith, "i", "")
	m := &systemstats{
		TotalMemory: string(totalmem),
		UsedMemory:  string(usedmem),
		TotalDisk:   string(totaldisk),
		UsedDisk:    string(useddisk),
		Bandwidth:   strings.TrimSuffix(stringBandwith, "\n"),
		Cores:       strings.TrimSuffix(string(cores), "\n"),
		Cpu:         strings.TrimSuffix(string(cpuname), "\n"),
		Os:          strings.TrimSuffix(string(os), "\n"),
		Uptime:      strings.TrimSuffix(string(uptime), "\n"),
		LoadAvg:     strings.TrimSuffix(string(loadavg), "\n"),
		CpuIdeal:    strings.TrimSuffix(string(cpuideal), "\n"),
	}
	return c.JSON(http.StatusOK, m)
}
