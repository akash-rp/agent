package main

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"github.com/panjf2000/ants/v2"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
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

var sslIssuer, _ = ants.NewPool(1, ants.WithNonblocking(true))

// var jobMap = make(map[string]*gocron.Job)

func main() {
	defer sslIssuer.Release()
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
	addRoutes(e)

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
	loadavg, _ := exec.Command("/bin/bash", "-c", "awk '{ printf \"%s %s %s\",$1,$2,$3}' /proc/loadavg").Output()
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

func linuxCommand(cmd string) ([]byte, error) {
	out, err := exec.Command("/bin/bash", "-c", cmd).CombinedOutput()
	return out, err
}
