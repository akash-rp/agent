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
	log.Print("Initializing CronJob")
	cronInt.StartAsync()
	for _, site := range obj.Sites {
		addCronJob(site.LocalBackup, site.Name, site.User)
	}
}

func updateLocalBackup(c echo.Context) error {
	backupType := c.Param("type")
	name := c.Param("name")
	user := c.Param("user")
	backup := new(Backup)
	c.Bind(&backup)
	if backup.Automatic == true {
		if backupType == "new" {
			err := addNewBackup(name, user, *backup)
			if err != nil {
				return c.JSON(echo.ErrNotFound.Code, "Error adding new Backup")
			}
			return c.JSON(http.StatusOK, "")
		} else if backupType == "existing" {
			cronInt.RemoveByTag(fmt.Sprintf("%s", name))
			err := addCronJob(*backup, name, user)
			if err != nil {
				return c.JSON(echo.ErrNotFound.Code, "Error adding cron job")
			}
			for i, site := range obj.Sites {
				if site.Name == name {
					obj.Sites[i].LocalBackup = *backup
				}
			}
			back, _ := json.MarshalIndent(obj, "", "  ")
			ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
			return c.JSON(http.StatusOK, "")

		}
	} else {
		return c.JSON(http.StatusOK, "Nothing to change")
	}
	return c.JSON(echo.ErrNotFound.Code, "Invalid Request")

}

func addNewBackup(name string, user string, backup Backup) error {
	found := false
	for i, site := range obj.Sites {
		if site.Name == name {
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository create filesystem --path='/var/Backup/%s' --password=%s ; kopia policy set --keep-latest 10 --keep-hourly %s --keep-daily %s --keep-weekly %s --keep-monthly %s --keep-annual 0 --global; kopia snapshot create /home/%s/%s ", name, name, backup.Retention.Hourly, backup.Retention.Daily, backup.Retention.Weekly, backup.Retention.Monthly, user, name)).CombinedOutput()
			if err != nil {
				log.Print(err)
				return err
			}
			obj.Sites[i].LocalBackup = backup
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
	f, err := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/backup.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
	}
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("Backup Process started\n"))
	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/%s --password=%s ; kopia snapshot create /home/%s/%s", name, name, user, name)).Output()
	if err == nil {
		f.Write([]byte("Backup Process Completed\n"))
	} else {
		f.Write([]byte("Backup Process Failed"))
		f.Write([]byte(err.Error()))

	}
	if err := f.Close(); err != nil {
		log.Print("Cannot write to backup log")
	}

}

func addCronJob(backup Backup, name string, user string) error {
	var err error
	if (backup != Backup{}) {
		if backup.Automatic {
			switch backup.Frequency {
			case "Hourly":
				_, err = cronInt.Cron(fmt.Sprintf("%s * * * *", backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user)
				})
				if err != nil {
					log.Print(err)
				}
				err = cronInt.RunByTag(name)
				if err != nil {
					log.Print(err)
				}
				return nil

			case "Daily":
				if _, err = cronInt.Every(1).Day().At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user)
				}); err != nil {
					log.Print(err)
				}
				return nil
			case "Weekly":
				switch backup.Time.WeekDay {
				case "Sunday":
					if _, err = cronInt.Every(1).Weekday(time.Sunday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
				case "Monday":
					if _, err = cronInt.Every(1).Weekday(time.Monday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
				case "Tuesday":
					if _, err = cronInt.Every(1).Weekday(time.Tuesday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
				case "Wednesday":
					if _, err = cronInt.Every(1).Weekday(time.Wednesday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
				case "Thursday":
					if _, err = cronInt.Every(1).Weekday(time.Thursday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
				case "Friday":
					if _, err = cronInt.Every(1).Weekday(time.Friday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
				case "Saturday":
					if _, err = cronInt.Every(1).Weekday(time.Saturday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
				}

			case "Monthly":
				day, _ := strconv.Atoi(backup.Time.MonthDay)
				if _, err = cronInt.Every(1).Month(day).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user)
				}); err != nil {
					log.Print(err)
				}
			}
		}

	}
	if err != nil {
		return err
	}
	return nil
}
