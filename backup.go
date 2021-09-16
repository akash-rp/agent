package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func initCron() {
	fmt.Print(obj)
	for _, site := range obj.Sites {
		addCronJob(site.LocalBackup, site.Name, site.User)
	}
	cronInt.StartAsync()
}

func updateLocalBackup(c echo.Context) error {
	backupType := c.Param("type")
	name := c.Param("name")
	user := c.Param("user")
	backup := new(Backup)
	c.Bind(&backup)

	if backupType == "new" {
		err := addNewBackup(name, user, *backup)
		if err != nil {
			return c.JSON(echo.ErrNotFound.Code, err)
		}
		return c.JSON(http.StatusOK, "")
	} else if backupType == "existing" {
		cronInt.RemoveByTag(fmt.Sprintf("%s", name))
		err := addCronJob(*backup, name, user)
		if err != nil {
			return c.JSON(echo.ErrNotFound.Code, err)
		}
		for _, site := range obj.Sites {
			if site.Name == name {
				site.LocalBackup = *backup
			}
		}
		back, _ := json.MarshalIndent(obj, "", "  ")
		ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
		return c.JSON(http.StatusOK, "")

	}
	return c.JSON(echo.ErrNotFound.Code, "Invalid Request")

}

func addNewBackup(name string, user string, backup Backup) error {
	found := false
	for _, site := range obj.Sites {
		if site.Name == name {
			_, err := exec.Command(fmt.Sprintf("kopia repository create filesystem --path='/var/Backup/%s' --password=%s ; kopia policy set --keep-latest 10 --keep-hourly %s --keep-daily %s --keep-weekly %s --keep-monthly %s --keep-yearly 0 --global; kopia snapshot create /home/%s/%s ", name, name, backup.Retention.Hourly, backup.Retention.Daily, backup.Retention.Weekly, backup.Retention.Monthly, user, name)).Output()
			if err != nil {
				return err
			}
			site.LocalBackup = backup
			back, _ := json.MarshalIndent(obj, "", "  ")
			ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
			found = true
			err = addCronJob(backup, name, user)
			if err != nil {
				return err
			}
		}
	}
	if found {
		return nil
	} else {
		return errors.New("Site not found")
	}
}

func takeBackup(name string, user string) {
	f, _ := os.OpenFile(fmt.Sprintf("/var/log/Hosting/%s", name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("Backup Process started\n"))
	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	_, err := exec.Command(fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/%s --password=%s ; kopia snapshot create /home/%s/%s", name, name, user, name)).Output()
	if err == nil {
		f.Write([]byte("Backup Process Completed\n"))
	} else {
		f.Write([]byte("Backup Process Failed"))
		f.Write([]byte(err.Error()))

	}

	if err := f.Close(); err != nil {
		log.Fatal(err)
	}

}

func addCronJob(backup Backup, name string, user string) error {
	var err error
	if (backup != Backup{}) {
		if backup.Automatic {
			switch backup.Frequency {
			case "Hourly":
				_, err = cronInt.Cron(fmt.Sprintf("%d * * * *", backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user)
				})
				cronInt.RunByTag(name)

			case "Daily":
				_, err = cronInt.Every(1).Day().At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
					takeBackup(name, user)
				})
			case "Weekly":
				switch backup.Time.WeekDay {
				case "Sunday":
					_, err = cronInt.Every(1).Weekday(time.Sunday).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
						takeBackup(name, user)
					})
				case "Monday":
					_, err = cronInt.Every(1).Weekday(time.Monday).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
						takeBackup(name, user)
					})
				case "Tuesday":
					_, err = cronInt.Every(1).Weekday(time.Tuesday).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
						takeBackup(name, user)
					})
				case "Wednesday":
					_, err = cronInt.Every(1).Weekday(time.Wednesday).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
						takeBackup(name, user)
					})
				case "Thursday":
					_, err = cronInt.Every(1).Weekday(time.Thursday).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
						takeBackup(name, user)
					})
				case "Friday":
					_, err = cronInt.Every(1).Weekday(time.Friday).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
						takeBackup(name, user)
					})
				case "Saturday":
					_, err = cronInt.Every(1).Weekday(time.Saturday).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
						takeBackup(name, user)
					})
				}

			case "Monthly":
				day, _ := strconv.Atoi(backup.Time.MonthDay)
				_, err = cronInt.Every(1).Month(day).At(fmt.Sprintf("%s", backup.Time)).Tag(name).Do(func() {
					takeBackup(name, user)
				})
			}
		}

	}
	if err != nil {
		return err
	}
	return nil
}
