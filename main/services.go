package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func getServiceStatus(c echo.Context) error {
	service := new(struct {
		Lsws      bool `json:"lsws"`
		Mariadb   bool `json:"mariadb"`
		Memcached bool `json:"memcached"`
		Redis     bool `json:"redis"`
		Newrelic  bool `json:"newrelic"`
	})
	status, _ := exec.Command("/bin/bash", "-c", "systemctl check lsws mariadb lsmcd redis newrelic-daemon").Output()
	statusArray := strings.Split(string(status), "\n")
	if statusArray[0] == "active" {
		service.Lsws = true
	} else {
		service.Lsws = false
	}
	if statusArray[1] == "active" {
		service.Mariadb = true
	} else {
		service.Mariadb = false
	}
	if statusArray[2] == "active" {
		service.Memcached = true
	} else {
		service.Memcached = false
	}
	if statusArray[3] == "active" {
		service.Redis = true
	} else {
		service.Redis = false
	}
	if statusArray[4] == "active" {
		service.Newrelic = true
	} else {
		service.Newrelic = false
	}
	return c.JSON(200, service)
}

func serviceRestart(c echo.Context) error {
	service := c.Param("service")
	serviceList := []string{"lsws", "mariadb", "redis", "lsmcd", "newrelic-daemon"}
	for _, ser := range serviceList {
		if ser == service {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s restart", ser)).Output()
			if err != nil {
				return c.JSON(400, "Service restart failed")
			} else {
				return c.JSON(200, "Success")
			}
		}
	}
	return c.JSON(400, "Service not found")
}

func serviceStop(c echo.Context) error {
	service := c.Param("service")
	serviceList := []string{"redis", "lsmcd", "newrelic"}
	for _, ser := range serviceList {
		if ser == service {
			if ser == "newrelic" {
				disableNewrelic()
				return c.JSON(200, "success")
			}
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s stop", ser)).Output()
			if err != nil {
				return c.JSON(400, "Failed to stop service")
			} else {
				return c.JSON(200, "Success")
			}
		}
	}
	return c.JSON(400, "Service not found")
}

func serviceStart(c echo.Context) error {
	service := c.Param("service")
	serviceList := []string{"redis", "lsmcd"}
	for _, ser := range serviceList {
		if ser == service {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s start", ser)).Output()
			if err != nil {
				return c.JSON(400, "Failed to start service")
			} else {
				return c.JSON(200, "Success")
			}
		}
	}
	return c.JSON(400, "Service not found")
}
