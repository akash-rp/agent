package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/sethvargo/go-password/password"
)

func createStaging(c echo.Context) error {
	Name := c.Param("name")
	User := c.Param("user")
	Mode := c.Param("mode")
	Url := c.Param("url")
	LivesiteUrl := c.Param("livesiteurl")
	logFile, _ := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/staging.log", Name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	logFile.Write([]byte("------------------------------------------------------------------------------\n"))
	logFile.Write([]byte("Starting Staging process\n"))
	logFile.Write([]byte("Time:" + time.Now().String() + "\n"))
	logFile.Write([]byte("Taking ondemad backup of Live site\n"))
	err := takeLocalOndemandBackup(Name, Mode, User, true)
	if err != nil {
		logFile.Write([]byte("Error occured while taking backup \n"))
		logFile.Write([]byte("Staging process failed\n. Exiting"))
		logFile.Close()
		return c.JSON(http.StatusBadRequest, "Backup process Failed")
	}
	logFile.Write([]byte(fmt.Sprintf("Copying file and folders from %s to %s_Staging\n", Name, Name)))
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("cp -r /home/%s/%s /home/%s/%s_Staging", User, Name, User, Name)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Error occured while copying files \n"))
		logFile.Write([]byte(out))
		logFile.Write([]byte("Staging process failed\n. Exiting"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to copy files")
	}
	logFile.Write([]byte("Taking Database Dump\n"))
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/wp-config.php | grep DB_NAME | cut -d \\' -f 4", User, Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 {
		logFile.Write([]byte("Invalid wp-config file configuration\n"))
		logFile.Write([]byte("Staging process Failed\n. Exiting"))
		logFile.Close()
		return errors.New("invalid wp-config file")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -B %s -o /home/%s/%s/DatabaseBackup/", dbnameArray[0], User, Name)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Failed to create database dump"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return errors.New("database Dump error")
	}
	logFile.Write([]byte("Restoring Database dump to staging database\n"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -d /home/%s/%s/DatabaseBackup -o -B %s_Staging", User, Name, Name)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Failed to create staging database"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()

		return c.JSON(echo.ErrNotFound.Code, "Failed to create staging database")
	}
	logFile.Write([]byte("Performing database search and replace opteration\n"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s_Staging -u root -p '' -s http://%s -r http://%s -x guid -x user_email", Name, LivesiteUrl, Url)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Failed to create staging database"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return errors.New("search and replace operation failed")
	}
	pass, _ := password.Generate(32, 20, 0, false, true)
	logFile.Write([]byte("creating new user and granting access to staging database\n"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"CREATE USER '%s_Staging'@'localhost' IDENTIFIED BY '%s';\"", Name, pass)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Failed to create staging database user\n"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to create staging user DB")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"GRANT ALL PRIVILEGES ON %s_Staging.* TO '%s_Staging'@'localhost';\"", Name, Name)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Failed to grant privileges to the db\n"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to grant privileges")
	}
	exec.Command("/bin/bash", "-c", "mysql -e 'FLUSH PRIVILEGES;'").Output()
	logFile.Write([]byte("Replacing wp-config file of staging site with new credentials\n"))
	path := fmt.Sprintf("/home/%s/%s_Staging/wp-config.php", User, Name)
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_NAME.*/define( \\'\\'DB_NAME\\'\\\\', \\'\\'%s_Staging\\'\\\\');/' %s", Name, path)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Failed to modify DB_NAME"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_USER.*/define( \\'\\'DB_USER\\'\\\\', \\'\\'%s_Staging\\'\\\\');/' %s", Name, path)).CombinedOutput()
	if err != nil {
		logFile.Write([]byte("Failed to modify DB_USER\n"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_PASSWORD.*/define( \\'\\'DB_PASSWORD\\'\\\\', \\'\\'%s\\'\\\\');/' %s", pass, path)).CombinedOutput()

	if err != nil {
		logFile.Write([]byte("Failed to modify DB_PASSWORD\n"))
		logFile.Write([]byte(string(out)))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	lsws := wpadd{AppName: Name + "_Staging", UserName: User, Url: Url}
	logFile.Write([]byte("Adding site to openlitespeed vhosts\n"))
	err = editLsws(lsws)
	if err != nil {
		logFile.Write([]byte("Failed to add vhost\n"))
		logFile.Write([]byte(string(err.Error())))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to add vhost")
	}
	logFile.Write([]byte("Adding site to proxy\n"))
	err = addSiteToJSON(lsws, "staging")
	if err != nil {
		logFile.Write([]byte("Failed to add site\n"))
		logFile.Write([]byte(string(err.Error())))
		logFile.Write([]byte("Staging Process Failed\n"))
		logFile.Close()
		return c.JSON(echo.ErrBadRequest.Code, "Failed to add site to proxy")
	}
	configNuster()
	go exec.Command("/bin/bash", "-c", "service lsws restart")
	go exec.Command("/bin/bash", "-c", "service hosting restart")
	return c.JSON(200, "Success")
}

func getDatabaseTables(c echo.Context) error {
	Name := c.Param("name")
	User := c.Param("user")
	var tables []string
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/wp-config.php | grep DB_NAME | cut -d \\' -f 4", User, Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 || len(dbnameArray) == 0 {
		return errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/wp-config.php | grep DB_USER | cut -d \\' -f 4", User, Name)).Output()
	dbuser := strings.TrimSuffix(string(db), "\n")
	dbuserArray := strings.Split(dbuser, "\n")
	if len(dbuserArray) > 1 || len(dbuserArray) == 0 {

		return errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/wp-config.php | grep DB_PASSWORD | cut -d \\' -f 4", User, Name)).Output()
	dbpassword := strings.TrimSuffix(string(db), "\n")
	dbpasswordArray := strings.Split(dbpassword, "\n")
	if len(dbpasswordArray) > 1 || len(dbpasswordArray) == 0 {

		return errors.New("invalid wp-config file")
	}
	mysql, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s"))
	if err != nil {
		log.Fatal(err)
	}
	defer mysql.Close()
	err = mysql.Ping()
	if err != nil {
		log.Fatal(err)
	}
	rows, err := mysql.Query("show tables")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		if err != nil {
			log.Fatal(err)
		}
		tables = append(tables, name)
		// log.Println(name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	// j, _ := json.Marshal(tables)
	return c.JSON(http.StatusOK, tables)
}
