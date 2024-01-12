package main

import (
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func getServiceStatus(c echo.Context) error {

	service := servicesStatus()
	return c.JSON(200, service)
}

func isServiceActive(status string) bool {
	return status == "active"
}

func singleServiceStatus(serviceName, processName, status string) SingleService {
	return SingleService{
		Service: serviceName,
		Running: isServiceActive(status),
		Process: processName,
	}
}

func servicesStatus() []SingleService {
	cmd := exec.Command("/bin/bash", "-c", "systemctl check lsws lsws mariadb lsmcd redis newrelic-daemon")
	output, _ := cmd.Output()
	statusArray := strings.Split(strings.TrimSpace(string(output)), "\n")

	var serviceList []SingleService

	var services = []struct {
		name    string
		process string
		status  string
	}{
		{"Litespeed", "lsws", ""},
		{"PHP", "lsphp", ""},
		{"Mariadb", "mariadb", ""},
		{"Memcached", "lsmcd", ""},
		{"Redis", "redis", ""},
		{"New Relic", "newrelic-daemon", ""},
	}

	for i, service := range services {
		serviceList = append(serviceList, singleServiceStatus(service.name, service.process, statusArray[i]))
	}

	return serviceList
}

func manageService(c echo.Context) error {
	service := new(ServiceAction)

	c.Bind(&service)

	switch service.Action {
	case "restart":
		if service.Service == "lsphp" {
			exec.Command("/bin/bash", "-c", "killall lsphp").Output()
		} else {
			if err := restartService(service.Service); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}
		return c.JSON(http.StatusOK, servicesStatus())

	case "stop":
		if service.Service == "newrelic-daemon" {
			disableNewrelic()
		} else {
			if err := stopService(service.Service); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}
		return c.JSON(http.StatusOK, servicesStatus())

	case "start":
		if service.Service == "newrelic-daemon" {
			enableNewrelic(0)
		} else {
			if err := startService(service.Service); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}
		}
		return c.JSON(http.StatusOK, servicesStatus())

	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid action")
	}
}

func restartService(service string) error {
	serviceList := []string{"lsws", "mariadb", "redis", "lsmcd", "newrelic-daemon"}
	fmt.Print(service)
	for _, ser := range serviceList {
		if ser == service {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s restart", ser)).Output()
			if err != nil {
				return errors.New("failed to restart service")
			}
			return nil
		}
	}
	return errors.New("service not found")
}

func stopService(service string) error {
	serviceList := []string{"redis", "lsmcd", "newrelic-daemon"}
	for _, ser := range serviceList {
		if ser == service {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s stop", ser)).Output()
			if err != nil {
				return errors.New("failed to stop service")
			}
			return nil
		}
	}
	return errors.New("service not found")
}

func startService(service string) error {
	serviceList := []string{"lsws", "mariadb", "redis", "lsmcd"}
	for _, ser := range serviceList {
		if ser == service {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service %s start", ser)).Output()
			if err != nil {
				fmt.Print("error occurred while starting service")
				return errors.New("failed to start service")
			}
			return nil
		}
	}
	return errors.New("service not found")
}
