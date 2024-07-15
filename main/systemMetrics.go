package main

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nakabonne/tstorage"
)

func storeMetrics() {

	usedMemString, _ := exec.Command("/bin/bash", "-c", "free -m | awk 'NR==2{printf $3}'").Output()
	idealCpuString, _ := exec.Command("/bin/bash", "-c", "echo $[100-$(vmstat 1 2|tail -1|awk '{print $15}')]").Output()
	loadAvgString, _ := exec.Command("/bin/bash", "-c", "uptime | awk -F'[, ]' '{print $17}'").Output()
	usedDiskString, _ := exec.Command("/bin/bash", "-c", "df -h --total -x tmpfs -x devtmpfs -x udev| awk '/total/{printf $3}' | awk '{print substr($1, 1, length($1)-1)}'").Output()
	// log.Print(string(idealCpuString), string(loadAvgString), string(usedDiskString))

	usedMem, _ := strconv.ParseFloat(strings.TrimSuffix(string(usedMemString), "\n"), 64)
	idealCpu, _ := strconv.ParseFloat(strings.TrimSuffix(string(idealCpuString), "\n"), 64)
	loadAvg, _ := strconv.ParseFloat(strings.TrimSuffix(string(loadAvgString), "\n"), 64)
	usedDisk, err := strconv.ParseFloat(strings.TrimSuffix(string(usedDiskString), "\n"), 64)
	if err != nil {
		log.Print(err)
	}
	now := time.Now().UnixMilli()
	if metrics == nil {
		metrics, err = tstorage.NewStorage(
			tstorage.WithTimestampPrecision(tstorage.Milliseconds),
			tstorage.WithDataPath("/usr/Hosting/metrics"),
			tstorage.WithWALBufferedSize(0),
		)
		if err != nil {
			log.Print(err.Error())
		}
	}
	if metrics != nil {

		metrics.InsertRows([]tstorage.Row{
			{
				Metric:    "Memory",
				DataPoint: tstorage.DataPoint{Timestamp: now, Value: usedMem},
			},
			{
				Metric:    "Cpu",
				DataPoint: tstorage.DataPoint{Timestamp: now, Value: idealCpu},
			},
			{
				Metric:    "Load",
				DataPoint: tstorage.DataPoint{Timestamp: now, Value: loadAvg},
			},
			{
				Metric:    "Disk",
				DataPoint: tstorage.DataPoint{Timestamp: now, Value: usedDisk},
			},
		})

	} else {
		log.Print("Metrics pointer is nil.")
	}
}

func metricsPartation() {
	log.Print("Running Metrics Partation")
	if metrics != nil {
		err := metrics.Close()
		if err != nil {
			log.Print(err.Error())
		}
	}
	var err error
	metrics, err = tstorage.NewStorage(
		tstorage.WithTimestampPrecision(tstorage.Milliseconds),
		tstorage.WithDataPath("/usr/Hosting/metrics"),
		tstorage.WithWALBufferedSize(0),
	)
	if err != nil {
		log.Print(err.Error())
	}
	log.Print("Partation Completed")
}

func getAllMetrics(c echo.Context) error {
	period := c.Param("period")
	var minTime int

	switch period {
	case "1hr":
		minTime = 3600000
	case "3hr":
		minTime = 10800000
	case "6hr":
		minTime = 21600000
	case "12hr":
		minTime = 43200000
	case "1day":
		minTime = 86400000
	case "3days":
		minTime = 259200000
	case "7days":
		minTime = 604800000
	case "14days":
		minTime = 1209600000
	default:
		minTime = 3600000
	}

	metricsLog := new(struct {
		Memory []MetricsValue `json:"memory"`
		Cpu    []MetricsValue `json:"cpu"`
		Load   []MetricsValue `json:"load"`
		Disk   []MetricsValue `json:"disk"`
	})

	memory, _ := metrics.Select("Memory", nil, time.Now().UnixMilli()-int64(minTime), time.Now().UnixMilli())
	cpu, _ := metrics.Select("Cpu", nil, time.Now().UnixMilli()-int64(minTime), time.Now().UnixMilli())
	load, _ := metrics.Select("Load", nil, time.Now().UnixMilli()-int64(minTime), time.Now().UnixMilli())
	disk, _ := metrics.Select("Disk", nil, time.Now().UnixMilli()-int64(minTime), time.Now().UnixMilli())

	for _, points := range memory {
		metricsLog.Memory = append(metricsLog.Memory, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	for _, points := range cpu {
		metricsLog.Cpu = append(metricsLog.Cpu, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	for _, points := range load {
		metricsLog.Load = append(metricsLog.Load, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	for _, points := range disk {
		metricsLog.Disk = append(metricsLog.Disk, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}

	return c.JSON(200, metricsLog)
}

func getMetrics(c echo.Context) error {
	metric := c.Param("metric")
	period := c.Param("period")
	var Mintime int
	switch period {
	case "1hr":
		Mintime = 3600000
	case "3hr":
		Mintime = 10800000
	case "6hr":
		Mintime = 21600000
	case "12hr":
		Mintime = 43200000
	case "1day":
		Mintime = 86400000
	case "3days":
		Mintime = 259200000
	case "7days":
		Mintime = 604800000
	case "14days":
		Mintime = 1209600000
	}

	result, _ := metrics.Select(metric, nil, time.Now().UnixMilli()-int64(Mintime), time.Now().UnixMilli())

	resultArray := []MetricsValue{}
	for _, points := range result {
		resultArray = append(resultArray, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	return c.JSON(200, resultArray)
}
