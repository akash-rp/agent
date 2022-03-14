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

}

func metricsPartation() {
	metrics.Close()
	metrics, _ = tstorage.NewStorage(
		tstorage.WithTimestampPrecision(tstorage.Seconds),
		tstorage.WithDataPath("/usr/Hosting/metrics"),
		tstorage.WithWALBufferedSize(0),
	)
}

func getAllMetrics(c echo.Context) error {
	MetricsLog := new(struct {
		Memory []MetricsValue `json:"memory"`
		Cpu    []MetricsValue `json:"cpu"`
		Load   []MetricsValue `json:"load"`
		Disk   []MetricsValue `json:"disk"`
	})

	memory, _ := metrics.Select("Memory", nil, time.Now().UnixMilli()-3600000, time.Now().UnixMilli())
	cpu, _ := metrics.Select("Cpu", nil, time.Now().UnixMilli()-3600000, time.Now().UnixMilli())
	load, _ := metrics.Select("Load", nil, time.Now().UnixMilli()-3600000, time.Now().UnixMilli())
	disk, _ := metrics.Select("Disk", nil, time.Now().UnixMilli()-3600000, time.Now().UnixMilli())
	for _, points := range memory {
		MetricsLog.Memory = append(MetricsLog.Memory, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	for _, points := range cpu {
		MetricsLog.Cpu = append(MetricsLog.Cpu, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	for _, points := range load {
		MetricsLog.Load = append(MetricsLog.Load, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	for _, points := range disk {
		MetricsLog.Disk = append(MetricsLog.Disk, MetricsValue{Value: float64(points.Value), Time: int(points.Timestamp)})
	}
	return c.JSON(200, MetricsLog)

}

func getMetrice(c echo.Context) error {
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
