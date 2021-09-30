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
	"strings"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/labstack/echo/v4"
)

var cronBusy bool

func initCron() {
	cronInt.SetMaxConcurrentJobs(1, gocron.WaitMode)
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
			latest := getLatest(*backup)
			log.Print("After function: " + strconv.Itoa(latest))
			exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path='/var/Backup/auto/%s' --password=%s ; kopia policy set --keep-latest %d --keep-hourly 0 --keep-daily 0 --keep-weekly 0 --keep-monthly 0 --keep-annual 0 --global;", name, name, latest)).Output()
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
	latest := getLatest(backup)

	for i, site := range obj.Sites {
		if site.Name == name {
			if latest == 0 {
				return errors.New("Error in retention policy")
			}
			output, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository create filesystem --path='/var/Backup/auto/%s' --password=%s ; kopia policy set --keep-latest %d --keep-hourly 0 --keep-daily 0 --keep-weekly 0 --keep-monthly 0 --keep-annual 0 --global", name, name, latest)).CombinedOutput()
			if err != nil {
				log.Print(string(output))
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
	cronBusy = true
	f, err := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/backup.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
	}
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("Backup Process started\n"))
	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	f.Write([]byte(user))
	f.Write([]byte(name))
	db, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/wp-config.php | grep DB_NAME | cut -d \\' -f 4", user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to DB Name"))
		f.Write([]byte(db))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return
	}
	f.Write([]byte(string(db)))
	dbname := strings.TrimSuffix(string(db), "\n")
	f.Write([]byte(string(dbname)))

	dbnameArray := strings.Split(dbname, "\n")
	f.Write([]byte(string(dbnameArray[0])))

	if len(dbnameArray) > 1 {
		f.Write([]byte("Invalid wp-config file configuration\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return
	}
	dbout, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -B %s -o /home/%s/%s/DatabaseBackup/", dbnameArray[0], user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create database backup"))
		f.Write([]byte(string(dbout)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/DatabaseBackup/metadata", user, name)).Output()
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/auto/%s --password=%s ; kopia snapshot create /home/%s/%s", name, name, user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Cannot create backup"))
		f.Write([]byte(string(out)))

	}
	for i, site := range obj.Sites {
		if site.Name == name {
			obj.Sites[i].LocalBackup.LastRun = time.Now().UTC().Format(time.RFC3339)
		}
	}
	err = SaveJSONFile()
	if err != nil {
		f.Write([]byte("Cannot save config file"))
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s/DatabaseBackup/", user, name)).Output()
	if err == nil {
		f.Write([]byte("Backup Process Completed\n"))
		cronBusy = false
	} else {
		f.Write([]byte("Backup Process Failed"))
		f.Write([]byte(fmt.Sprintf("%s%v", err, err)))
		f.Write([]byte(out))
		cronBusy = false
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
				if !previousBackupExecuted(backup.LastRun, "Hourly") {

					err = cronInt.RunByTag(name)
					if err != nil {
						log.Print(err)
					}
				}
				return nil

			case "Daily":
				if _, err = cronInt.Every(1).Day().At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user)
				}); err != nil {
					log.Print(err)
				}
				if !previousBackupExecuted(backup.LastRun, "Daily") {

					err = cronInt.RunByTag(name)
					if err != nil {
						log.Print(err)
					}
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
					if !previousBackupExecuted(backup.LastRun, "Weekly") {

						err = cronInt.RunByTag(name)
						if err != nil {
							log.Print(err)
						}
					}
				case "Monday":
					if _, err = cronInt.Every(1).Weekday(time.Monday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(backup.LastRun, "Weekly") {

						err = cronInt.RunByTag(name)
						if err != nil {
							log.Print(err)
						}
					}
				case "Tuesday":
					if _, err = cronInt.Every(1).Weekday(time.Tuesday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(backup.LastRun, "Weekly") {

						err = cronInt.RunByTag(name)
						if err != nil {
							log.Print(err)
						}
					}
				case "Wednesday":
					if _, err = cronInt.Every(1).Weekday(time.Wednesday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(backup.LastRun, "Weekly") {

						err = cronInt.RunByTag(name)
						if err != nil {
							log.Print(err)
						}
					}
				case "Thursday":
					if _, err = cronInt.Every(1).Weekday(time.Thursday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(backup.LastRun, "Weekly") {

						err = cronInt.RunByTag(name)
						if err != nil {
							log.Print(err)
						}
					}
				case "Friday":
					if _, err = cronInt.Every(1).Weekday(time.Friday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(backup.LastRun, "Weekly") {

						err = cronInt.RunByTag(name)
						if err != nil {
							log.Print(err)
						}
					}
				case "Saturday":
					if _, err = cronInt.Every(1).Weekday(time.Saturday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user)
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(backup.LastRun, "Weekly") {

						err = cronInt.RunByTag(name)
						if err != nil {
							log.Print(err)
						}
					}
				}

			case "Monthly":
				day, _ := strconv.Atoi(backup.Time.MonthDay)
				if _, err = cronInt.Every(1).Month(day).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user)
				}); err != nil {
					log.Print(err)
				}
				if !previousBackupExecuted(backup.LastRun, "Weekly") {

					err = cronInt.RunByTag(name)
					if err != nil {
						log.Print(err)
					}
				}
			}
		}

	}
	if err != nil {
		return err
	}
	return nil
}

func nextrun(c echo.Context) error {
	_, time := cronInt.NextRun()
	return c.JSON(http.StatusOK, time)
}

func getLatest(backup Backup) int {
	latest := 0
	log.Print(backup.Frequency)
	log.Print(backup.Retention.Type)
	log.Print(backup.Retention.Time)
	switch backup.Frequency {
	case "Hourly":
		switch backup.Retention.Type {
		case "Day":
			log.Print("Entered Day case")
			latest = 24 * backup.Retention.Time
			log.Print("Latest: " + strconv.Itoa(latest))
		case "Week":
			latest = 24 * 7 * backup.Retention.Time
		case "Month":
			latest = 24 * 28 * backup.Retention.Time
		}
	case "Daily":
		switch backup.Retention.Type {
		case "Day":
			latest = 1 * backup.Retention.Time
		case "Week":
			latest = 7 * backup.Retention.Time
		case "Month":
			latest = 28 * backup.Retention.Time
		}
	case "Weekly":
		switch backup.Retention.Type {
		case "Week":
			latest = 1 * backup.Retention.Time
		case "Month":
			latest = 4 * backup.Retention.Time
		}
	case "Monthly":
		switch backup.Retention.Type {
		case "Month":
			latest = 1 * backup.Retention.Time
		}
	}
	return latest
}

func takeLocalBackup(c echo.Context) error {
	name := c.Param("name")
	backupType := c.Param("type")
	user := c.Param("user")
	cronBusy = true
	f, err := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/backup.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
	}
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("ONDEMAND Backup Process started\n"))
	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/wp-config.php | grep DB_NAME | cut -d \\' -f 4", user, name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 {
		f.Write([]byte("Invalid wp-config file configuration\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return c.JSON(http.StatusBadRequest, "Invalid wp-config file")
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -B %s -o /home/%s/%s/DatabaseBackup/", dbnameArray[0], user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create database backup"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return c.JSON(http.StatusBadRequest, "Database Dump error")

	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/DatabaseBackup/metadata", user, name)).Output()
	if backupType == "new" {
		out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository create filesystem --path=/var/Backup/ondemand/%s --password=%s ; kopia policy set --keep-latest 10 --keep-hourly 0 --keep-daily 0 --keep-weekly 0 --keep-monthly 0 --keep-annual 0 --global; kopia snapshot create /home/%s/%s", name, name, user, name)).CombinedOutput()
	}
	if backupType == "existing" {
		out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/ondemand/%s --password=%s ; kopia snapshot create /home/%s/%s", name, name, user, name)).CombinedOutput()
	}
	if err != nil {
		f.Write([]byte("Cannot create backup"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return c.JSON(http.StatusBadRequest, "Cannot create backup")

	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s/DatabaseBackup/", user, name)).Output()
	if err == nil {

		f.Write([]byte("Backup Process Completed\n"))
		f.Close()
		cronBusy = false
		return c.JSON(http.StatusOK, "")
	} else {

		f.Write([]byte("Backup Process Failed"))
		f.Write([]byte(err.Error()))
		f.Close()
		cronBusy = false
		return c.JSON(echo.ErrBadRequest.Code, "Cannot create Backup")
	}
	if err := f.Close(); err != nil {
		log.Print("Cannot write to backup log")
	}
	return c.JSON(http.StatusOK, "")
}

func getLocalBackupsList(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	backuptype := c.Param("mode")

	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}

	_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/%s/%s --password=%s", backuptype, name, name)).Output()
	if err != nil {
		return c.JSON(echo.ErrNotFound.Code, "Cannot connect to filesystem")
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia snapshot list /home/%s/%s --json --json-indent", user, name)).CombinedOutput()
	if err != nil {
		return c.JSON(echo.ErrNotFound.Code, "Cannot list backups")
	}
	return c.JSON(http.StatusOK, string(out))

}

func restoreBackup(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	id := c.Param("id")
	restoreType := c.Param("type")
	mode := c.Param("mode")
	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	if restoreType == "both" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/%s/%s --password=%s ; kopia restore %s /home/%s/%s", mode, name, name, id, user, name)).CombinedOutput()
		if err != nil {
			log.Print(out)

			return c.JSON(echo.ErrNotFound.Code, "Failed to Restore Backup from Backup System")
		}
		exec.Command("/bin/bash", "-c", fmt.Sprintf("touch /home/%s/%s/DatabaseBackup/metadata", user, name)).Output()
		out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -d /home/%s/%s/DatabaseBackup -o", user, name)).CombinedOutput()
		if err != nil {
			log.Print(out)

			return c.JSON(echo.ErrNotFound.Code, "Failed to Restore Database")
		}
		exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s/DatabaseBackup", user, name)).Output()
		return c.JSON(http.StatusOK, "Success")
	} else if restoreType == "db" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/%s/%s --password=%s ; kopia restore %s/DatabaseBackup /home/%s/%s/DatabaseBackup", mode, name, name, id, user, name)).CombinedOutput()
		if err != nil {
			log.Print(out)

			return c.JSON(echo.ErrNotFound.Code, "Failed to Restore Backup from Backup System")
		}
		exec.Command("/bin/bash", "-c", fmt.Sprintf("touch /home/%s/%s/DatabaseBackup/metadata", user, name)).Output()
		out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -d /home/%s/%s/DatabaseBackup -o", user, name)).CombinedOutput()
		if err != nil {
			log.Print(out)
			return c.JSON(echo.ErrNotFound.Code, "Failed to Restore Database")
		}
		exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s/DatabaseBackup", user, name)).Output()
		return c.JSON(http.StatusOK, "Success")
	} else if restoreType == "webapp" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/%s/%s --password=%s ; kopia restore %s /home/%s/%s", mode, name, name, id, user, name)).CombinedOutput()
		if err != nil {
			log.Print(out)

			return c.JSON(echo.ErrNotFound.Code, "Failed to Restore Backup from Backup System")
		}
		exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s/DatabaseBackup", user, name)).CombinedOutput()
		return c.JSON(http.StatusOK, "Success")
	}
	return nil
}

func previousBackupExecuted(t string, frequency string) bool {
	if t == "" {
		return false
	}
	switch frequency {
	case "Hourly":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := int(now.Sub(old).Minutes())
		if diff > 60 {
			return false
		}
		return true
	case "Daily":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := now.Sub(old).Hours()
		if diff > 24 {
			return false
		}
		return true
	case "Weekly":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := now.Sub(old).Hours()
		if diff > 168 {
			return false
		}
		return true
	case "Monthly":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := now.Sub(old).Hours()
		if diff > 648 {
			return false
		}
		return true
	}
	return false
}
