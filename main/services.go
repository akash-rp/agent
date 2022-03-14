package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func getServiceStatus(c echo.Context) error {

	service := serviceStatus()
	return c.JSON(200, service)
}

func serviceStatus() []SingleService {

	service := new(struct {
		Service []SingleService `json:"service"`
	})
	status, _ := exec.Command("/bin/bash", "-c", "systemctl check lsws mariadb lsmcd redis newrelic-daemon").Output()
	statusArray := strings.Split(string(status), "\n")
	if statusArray[0] == "active" {
		service.Service = append(service.Service, SingleService{Service: "Litespeed", Running: true, Process: "lsws"})
	} else {
		service.Service = append(service.Service, SingleService{Service: "Litespeed", Running: false, Process: "lsws"})

	}
	service.Service = append(service.Service, SingleService{Service: "PHP", Running: true, Process: "lsphp"})
	if statusArray[1] == "active" {
		service.Service = append(service.Service, SingleService{Service: "Mariadb", Running: true, Process: "mariadb"})

	} else {
		service.Service = append(service.Service, SingleService{Service: "Mariadb", Running: false, Process: "mariadb"})

	}
	if statusArray[2] == "active" {
		service.Service = append(service.Service, SingleService{Service: "Memcached", Running: true, Process: "lsmcd"})

	} else {
		service.Service = append(service.Service, SingleService{Service: "Memcached", Running: false, Process: "lsmcd"})
	}
	if statusArray[3] == "active" {
		service.Service = append(service.Service, SingleService{Service: "Redis", Running: true, Process: "redis"})
	} else {
		service.Service = append(service.Service, SingleService{Service: "Redis", Running: false, Process: "redis"})
	}
	if statusArray[4] == "active" {
		service.Service = append(service.Service, SingleService{Service: "New Relic", Running: true, Process: "newrelic-daemon"})
	} else {
		service.Service = append(service.Service, SingleService{Service: "New Relic", Running: false, Process: "newrelic-daemon"})
	}
	return service.Service
}

func serviceRestart(c echo.Context) error {
	service := c.Param("service")
	if service == "lsphp" {
		exec.Command("/bin/bash", "-c", "killall lsphp").Output()
		service := serviceStatus()
		return c.JSON(200, service)
	}
	serviceList := []string{"lsws", "mariadb", "redis", "lsmcd", "newrelic-daemon"}
	for _, ser := range serviceList {
		if ser == service {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s restart", ser)).Output()
			service := serviceStatus()
			if err != nil {
				return c.JSON(400, "Failed to restart service")
			} else {
				return c.JSON(200, service)
			}
		}
	}
	return c.JSON(400, "Service not found")
}

func serviceStop(c echo.Context) error {
	service := c.Param("service")
	serviceList := []string{"redis", "lsmcd", "newrelic-daemon"}
	for _, ser := range serviceList {
		if ser == service {
			if ser == "newrelic-daemon" {
				disableNewrelic()
				service := serviceStatus()
				return c.JSON(200, service)
			}
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s stop", ser)).Output()
			service := serviceStatus()

			if err != nil {
				return c.JSON(400, "Failed to stop service")
			} else {
				return c.JSON(200, service)
			}
		}
	}
	return c.JSON(400, "Service not found")
}

func serviceStart(c echo.Context) error {
	service := c.Param("service")
	if service == "newrelic-daemon" {
		enabelNewrelic(0)
		service := serviceStatus()
		return c.JSON(200, service)
	}
	serviceList := []string{"redis", "lsmcd"}
	for _, ser := range serviceList {
		if ser == service {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s start", ser)).Output()
			service := serviceStatus()

			if err != nil {
				return c.JSON(400, "Failed to start service")
			} else {
				return c.JSON(200, service)
			}
		}
	}
	return c.JSON(400, "Service not found")
}
