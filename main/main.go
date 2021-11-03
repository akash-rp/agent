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
)

var obj Config
var cronInt = gocron.NewScheduler(time.UTC)

// var jobMap = make(map[string]*gocron.Job)

func main() {
	e := echo.New()
	data, err := ioutil.ReadFile("/usr/Hosting/config.json")
	if err != nil {
		log.Fatal("Cannot read file")
	}
	json.Unmarshal(data, &obj)
	configNuster()
	initCron()
	e.GET("/serverstats", serverStats)
	e.POST("/wp/add", wpAdd)
	e.POST("/wp/delete", wpDelete)
	e.GET("/hositng", hosting)
	e.POST("/cert", cert)
	e.GET("/sites", getSites)
	e.POST("/domainedit", editDomain)
	e.POST("/changeprimary", changePrimary)
	e.POST("/changePHP", changePHP)
	e.GET("/getPHPini/:name", getPHPini)
	e.POST("/updatePHPini/:name", updatePHPini)
	e.POST("/updatelocalbackup/:type/:name/:user", updateLocalBackup)
	e.GET("/takelocalondemandbackup/:type/:name/:user", ondemadBackup)
	e.GET("/localbackup/nextrun", nextrun)
	e.GET("/localbackup/list/:name/:user/:mode", getLocalBackupsList)
	e.GET("/restorelocalbackup/:name/:user/:mode/:id/:type", restoreBackup)
	e.GET("/createstaging/:name/:user/:mode/:url/:livesiteurl", createStaging)
	e.GET("/getdbtables/:name/:user", getDatabaseTables)
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
	}
	return c.JSON(http.StatusOK, m)
}

func hosting(c echo.Context) error {
	err := configNuster()
	if err != nil {
		return err
	}
	go exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	return c.String(http.StatusOK, "Success")
}
