package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sethvargo/go-password/password"
)

func createStaging(c echo.Context) error {
	var staging = new(Staging)

	c.Bind(&staging)
	logFile, _ := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/staging.log"), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	logFile.Write([]byte("------------------------------------------------------------------------------\n"))
	logFile.Write([]byte("Starting Staging process\n"))
	logFile.Write([]byte("Time:" + time.Now().String() + "\n"))
	logFile.Write([]byte("Taking ondemad backup of Live site"))
	err := takeLocalBackup(staging.Name, staging.Type, staging.User, true)
	if err != nil {
		logFile.Write([]byte("Error occured while taking backup \n"))
		logFile.Write([]byte("Staging process failed. Exiting"))
		logFile.Close()
		return c.JSON(http.StatusBadRequest, "Backup process Failed")
	}
	logFile.Write([]byte(fmt.Sprintf("Copying file and folders from %s to %s_Staging", staging.Name, staging.Name)))
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("cp -r /home/%s/%s /home/%s/%s_Staging", staging.User, staging.Name, staging.User, staging.Name)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Error occured while copying files \n"))
		logFile.Write([]byte(out))
		logFile.Write([]byte("Staging process failed. Exiting"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to copy files")
	}
	logFile.Write([]byte("Taking Database Dump"))
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/wp-config.php | grep DB_NAME | cut -d \\' -f 4", staging.User, staging.Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 {
		logFile.Write([]byte("Invalid wp-config file configuration\n"))
		logFile.Write([]byte("Staging process Failed. Exiting"))
		logFile.Close()
		return errors.New("Invalid wp-config file")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -B %s -o /home/%s/%s/DatabaseBackup/", dbnameArray[0], staging.User, staging.Name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create database dump"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Staging Process Failed"))
		f.Close()
		return errors.New("Database Dump error")
	}
	logFile.Write([]byte("Restoring Database dump to staging database"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -d /home/%s/%s/DatabaseBackup -o -B %s_Staging", staging.User, staging.Name, staging.Name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create staging database"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Staging Process Failed"))
		f.Close()

		return c.JSON(echo.ErrNotFound.Code, "Failed to create staging database")
	}
	pass, _ := password.Generate(32, 20, 0, false, true)
	logFile.Write([]byte("creating new user and granting access to staging database"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"CREATE USER '%s_Staging'@'localhost' IDENTIFIED BY '%s';\"", staging.Name, pass)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create staging database user"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Staging Process Failed"))
		f.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to create staging user DB")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"GRANT ALL PRIVILEGES ON %s_Staging.* TO '%s_Staging'@'localhost';\"", staging.Name, staging.User)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to grant privileges to the db"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Staging Process Failed"))
		f.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to grant privileges")
	}
	exec.Command("/bin/bash", "-c", "mysql -e 'FLUSH PRIVILEGES;'").Output()
	logFile.Write([]byte("Replacing wp-config file of staging site with new credentials"))
	path := fmt.Sprintf("/home/%s/%s_Staging/wp-config.php", staging.User, staging.Name)
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/DB_NAME.*/define( 'DB_NAME', '%s_Staging' );' %s", staging.Name, path)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to modify DB_NAME"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Staging Process Failed"))
		f.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/DB_NAME.*/define( 'DB_USER', '%s_Staging' );' %s", staging.Name, path)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to modify DB_USER"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Staging Process Failed"))
		f.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/DB_NAME.*/define( 'DB_PASSWORD', '%s' );' %s", pass, path)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to modify DB_PASSWORD"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Staging Process Failed"))
		f.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	lsws := wpadd{AppName: staging.Name + "_Staging", UserName: staging.User, Url: staging.Url}
	logFile.Write([]byte("Adding site to openlitespeed vhosts"))
	err = editLsws(lsws)
	if err != nil {
		f.Write([]byte("Failed to add vhost"))
		f.Write([]byte(string(err.Error())))
		f.Write([]byte("Staging Process Failed"))
		f.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to add vhost")
	}
	logFile.Write([]byte("Adding site to proxy"))
	err = addSiteToJSON(lsws)
}
