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

var cronBusy bool

func updateLocalBackupReq(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	backup := new(Backup)
	c.Bind(&backup)
	err := updateLocalBackup(name, user, backup)
	if err != nil {
		return c.NoContent(400)
	}
	return c.NoContent(200)
}

func updateLocalBackup(name string, user string, backup *Backup) error {
	lastBackup := ""
	if backup.Automatic {
		cronInt.RemoveByTag(name)
		latest := getLatest(*backup)
		out, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/automatic/automatic.config policy set /home/%s/%s --keep-latest %d --keep-hourly 0 --keep-daily 0 --keep-weekly 0 --keep-monthly 0 --keep-annual 0 ", user, name, latest)).CombinedOutput()
		log.Println(string(out))
		for i, site := range obj.Sites {
			if site.Name == name {
				lastBackup = site.LocalBackup.LastRun
				err := addCronJob(*backup, name, user, lastBackup)
				if err != nil {
					return errors.New("Error adding cron job")
				}
				obj.Sites[i].LocalBackup = *backup
				if lastBackup == "" {
					obj.Sites[i].LocalBackup.LastRun = time.Now().UTC().Format(time.RFC3339)
				} else {
					obj.Sites[i].LocalBackup.LastRun = lastBackup
				}
			}
		}
		back, _ := json.MarshalIndent(obj, "", "  ")
		ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
		return nil

	} else {
		cronInt.RemoveByTag(name)
		for i, site := range obj.Sites {
			if site.Name == name {
				obj.Sites[i].LocalBackup.Automatic = false
			}
		}
		back, _ := json.MarshalIndent(obj, "", "  ")
		ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
		return nil

	}
	return errors.New("Invalid Request")
}

// func addNewBackup(name string, user string, backup Backup) error {
// 	found := false
// 	latest := getLatest(backup)
// 	log.Print("Before Site enter")
// 	log.Print("Name: " + name)
// 	log.Print("user: " + user)
// 	for i, site := range obj.Sites {
// 		log.Print("Enterend Sites Range")
// 		if site.Name == name {
// 			if latest == 0 {
// 				log.Print("Latest is 0")
// 				return errors.New("error in retention policy")
// 			}
// 			output, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository create filesystem --path='/var/Backup/auto/%s' --password=%s", name, name)).CombinedOutput()
// 			log.Print(string(output))
// 			if err != nil {
// 				log.Print(string(output))
// 				log.Print("Error in Kopia create")
// 				return err
// 			}
// 			output, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia policy set --keep-latest %d --keep-hourly 0 --keep-daily 0 --keep-weekly 0 --keep-monthly 0 --keep-annual 0 --global", latest)).CombinedOutput()
// 			log.Print(string(output))
// 			if err != nil {
// 				log.Print(string(output))
// 				log.Print("Error in Kopia policy")
// 				return err
// 			}
// 			obj.Sites[i].LocalBackup = backup
// 			obj.Sites[i].LocalBackup.Created = true
// 			back, _ := json.MarshalIndent(obj, "", "  ")
// 			ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
// 			found = true
// 			err = addCronJob(backup, name, user, "")
// 			if err != nil {
// 				log.Print(err)
// 				return err
// 			}
// 		}
// 	}
// 	if found {
// 		return nil
// 	} else {
// 		return errors.New("site not found")
// 	}
// }

func takeBackup(name string, user string, msg string) {
	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true
	f, err := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/backup.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
	}
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("Backup Process started\n"))
	f.Write([]byte(msg))
	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	f.Write([]byte(user))
	f.Write([]byte(name))
	dbname, _, _, err := getDbcredentials(user, name)

	if err != nil {
		f.Write([]byte(err.Error()))
		f.Write([]byte("Invalid wp-config file configuration\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return
	}
	rootPass, err := getMariadbRootPass()
	if err != nil {
		f.Write([]byte(err.Error()))
		f.Write([]byte("Root password not found\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return
	}
	dbout, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -u root -p %s -B %s -o /home/%s/%s/private/DatabaseBackup/", rootPass, dbname, user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create database backup"))
		f.Write([]byte(string(dbout)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/automatic/automatic.config snapshot create /home/%s/%s", user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Cannot create backup"))
		f.Write([]byte(string(out)))

	}
	for i, site := range obj.Sites {
		if site.Name == name {
			obj.Sites[i].LocalBackup.LastRun = time.Now().UTC().Format(time.RFC3339)
			log.Print(obj.Sites[i])
		}
	}
	err = SaveJSONFile()
	if err != nil {
		f.Write([]byte("Cannot save config file"))
	}
	deleteDatabaseDump(user, name)
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

func addCronJob(backup Backup, name string, user string, lastBackup string) error {
	log.Print(backup)
	var err error
	if (backup != Backup{}) {
		if backup.Automatic {
			switch backup.Frequency {
			case "Hourly":
				_, err = cronInt.Cron(fmt.Sprintf("%s * * * *", backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user, "Started by cron Do function")
				})
				if err != nil {
					log.Print(err)
				}
				log.Print("Next is Prevoious function")
				if !previousBackupExecuted(lastBackup, "Hourly") {

					takeBackup(name, user, "Started Due to no last run")
				}
				return nil

			case "Daily":
				if _, err = cronInt.Every(1).Day().At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user, "Started by cron Do function")
				}); err != nil {
					log.Print(err)
				}
				if !previousBackupExecuted(lastBackup, "Daily") {

					takeBackup(name, user, "Started Due to no last run")
				}
				return nil
			case "Weekly":
				switch backup.Time.WeekDay {
				case "Sunday":
					if _, err = cronInt.Every(1).Weekday(time.Sunday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user, "Started by cron Do function")
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(lastBackup, "Weekly") {

						takeBackup(name, user, "Started Due to no last run")
					}
				case "Monday":
					if _, err = cronInt.Every(1).Weekday(time.Monday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user, "Started by cron Do function")
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(lastBackup, "Weekly") {

						takeBackup(name, user, "Started Due to no last run")
					}
				case "Tuesday":
					if _, err = cronInt.Every(1).Weekday(time.Tuesday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user, "Started by cron Do function")
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(lastBackup, "Weekly") {

						takeBackup(name, user, "Started Due to no last run")
					}
				case "Wednesday":
					if _, err = cronInt.Every(1).Weekday(time.Wednesday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user, "Started by cron Do function")
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(lastBackup, "Weekly") {

						takeBackup(name, user, "Started Due to no last run")
					}
				case "Thursday":
					if _, err = cronInt.Every(1).Weekday(time.Thursday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user, "Started by cron Do function")
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(lastBackup, "Weekly") {

						takeBackup(name, user, "Started Due to no last run")
					}
				case "Friday":
					if _, err = cronInt.Every(1).Weekday(time.Friday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user, "Started by cron Do function")
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(lastBackup, "Weekly") {

						takeBackup(name, user, "Started Due to no last run")
					}
				case "Saturday":
					if _, err = cronInt.Every(1).Weekday(time.Saturday).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
						takeBackup(name, user, "Started by cron Do function")
					}); err != nil {
						log.Print(err)
					}
					if !previousBackupExecuted(lastBackup, "Weekly") {

						takeBackup(name, user, "Started Due to no last run")
					}
				}

			case "Monthly":
				day, _ := strconv.Atoi(backup.Time.MonthDay)
				if _, err = cronInt.Every(1).Month(day).At(fmt.Sprintf("%s:%s", backup.Time.Hour, backup.Time.Minute)).Tag(name).Do(func() {
					takeBackup(name, user, "Started by cron Do function")
				}); err != nil {
					log.Print(err)
				}
				if !previousBackupExecuted(lastBackup, "Weekly") {

					takeBackup(name, user, "Started Due to no last run")
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

func ondemadBackup(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	err := takeLocalOndemandBackup(name, user, false)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}
	list, err := LocalBackupsList(name, user)
	if err != nil {
		return c.JSON(200, "success")
	}
	return c.JSON(http.StatusOK, list)
}

func takeLocalOndemandBackup(name string, user string, staging bool) error {
	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true
	f, err := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/backup.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
	}
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("ONDEMAND Backup Process started\n"))
	if staging {
		f.Write([]byte("Process started for crceating staging site\n"))
	}
	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	dbname, _, _, err := getDbcredentials(user, name)
	if err != nil {
		f.Write([]byte("Invalid wp-config file configuration\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return errors.New("invalid wp-config file")
	}
	rootPass, err := getMariadbRootPass()
	if err != nil {
		f.Write([]byte(err.Error()))
		f.Write([]byte("Root password not found\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return errors.New("Root password not found")
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -u root -p %s -B %s -o /home/%s/%s/private/DatabaseBackup/", rootPass, dbname, user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create database backup"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return errors.New("database Dump error")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()

	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/ondemand/ondemand.config snapshot create /home/%s/%s", user, name)).CombinedOutput()

	if err != nil {
		f.Write([]byte("Cannot create backup"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return errors.New("cannot create backup")

	}
	deleteDatabaseDump(user, name)
	if err == nil {

		f.Write([]byte("Backup Process Completed\n"))
		f.Close()
		cronBusy = false
		return nil
	} else {

		f.Write([]byte("Backup Process Failed"))
		f.Write([]byte(err.Error()))
		f.Close()
		cronBusy = false
		return errors.New("cannot create Backup")
	}

}

//not yet finished
func takeSystemBackup(name string, user string) error {
	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true
	f, err := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/backup.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
	}
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("System Backup Process started\n"))

	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	dbname, _, _, err := getDbcredentials(user, name)
	if err != nil {
		f.Write([]byte("Invalid wp-config file configuration\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return errors.New("invalid wp-config file")
	}
	rootPass, err := getMariadbRootPass()
	if err != nil {
		f.Write([]byte(err.Error()))
		f.Write([]byte("Root password not found\n"))
		f.Write([]byte("Backup Failed"))
		f.Close()
		cronBusy = false
		return errors.New("root password not found")
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -u root -p %s -B %s -o /home/%s/%s/private/DatabaseBackup/", rootPass, dbname, user, name)).CombinedOutput()
	if err != nil {
		f.Write([]byte("Failed to create database backup"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return errors.New("database Dump error")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()

	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/ondemand --password=kopia ; kopia snapshot create /home/%s/%s", user, name)).CombinedOutput()

	if err != nil {
		f.Write([]byte("Cannot create backup"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return errors.New("cannot create backup")

	}
	deleteDatabaseDump(user, name)
	if err == nil {

		f.Write([]byte("Backup Process Completed\n"))
		f.Close()
		cronBusy = false
		return nil
	} else {

		f.Write([]byte("Backup Process Failed"))
		f.Write([]byte(err.Error()))
		f.Close()
		cronBusy = false
		return errors.New("cannot create Backup")
	}

}

func getLocalBackupsList(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	list, err := LocalBackupsList(name, user)
	if err != nil {
		return c.JSON(400, err.Error())
	}
	return c.JSON(200, list)
}

func LocalBackupsList(name string, user string) (LocalBackup, error) {

	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true

	ondemand, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/ondemand/ondemand.config snapshot list /home/%s/%s --json", user, name)).CombinedOutput()
	if err != nil {
		cronBusy = false

		return LocalBackup{}, errors.New("Cannot list backups")
	}

	automatic, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/automatic/automatic.config snapshot list /home/%s/%s --json", user, name)).CombinedOutput()
	if err != nil {
		cronBusy = false

		return LocalBackup{}, errors.New("Cannot list backups")
	}

	system, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/system/system.config snapshot list /home/%s/%s --json", user, name)).CombinedOutput()
	if err != nil {
		cronBusy = false

		return LocalBackup{}, errors.New("Cannot list backups")
	}
	list := new(LocalBackup)
	json.Unmarshal(ondemand, &list.Ondemand)
	json.Unmarshal(automatic, &list.Automatic)
	json.Unmarshal(system, &list.System)
	cronBusy = false

	return *list, nil

}

func restoreBackupFromPanel(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	id := c.Param("id")
	restoreType := c.Param("type")
	mode := c.Param("mode")
	err := restoreBackup(name, user, id, restoreType, mode)
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}
	return c.JSON(200, "success")
}

func restoreBackup(name string, user string, id string, restoreType string, mode string) error {

	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true
	rootPass, err := getMariadbRootPass()
	if err != nil {
		return errors.New("Root password not found")
	}
	if restoreType == "both" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/%[1]s/%[1]s.config restore %[2]s /home/%[3]s/%[4]s", mode, id, user, name)).CombinedOutput()
		if err != nil {
			cronBusy = false

			log.Print(out)

			return errors.New("failed to Restore Backup from Backup System")
		}
		exec.Command("/bin/bash", "-c", fmt.Sprintf("touch /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()
		out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -u root -p %s -d /home/%s/%s/private/DatabaseBackup -o", rootPass, user, name)).CombinedOutput()
		if err != nil {
			cronBusy = false

			log.Print(out)

			return errors.New("failed to Restore Database")
		}
		deleteDatabaseDump(user, name)
		return nil
	} else if restoreType == "db" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/%[1]s/%[1]s.config restore %[2]s/DatabaseBackup /home/%[3]s/%[4]s/private/DatabaseBackup", mode, id, user, name)).CombinedOutput()
		if err != nil {
			cronBusy = false

			log.Print(out)

			return errors.New("failed to Restore Backup from Backup System")
		}
		exec.Command("/bin/bash", "-c", fmt.Sprintf("touch /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()
		out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -u root -p %s -d /home/%s/%s/private/DatabaseBackup -o", rootPass, user, name)).CombinedOutput()
		if err != nil {
			cronBusy = false

			log.Print(out)
			return errors.New("failed to Restore Database")
		}
		deleteDatabaseDump(user, name)
		cronBusy = false

		return nil
	} else if restoreType == "webapp" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/%[1]s/%[1]s.config restore %[2]s /home/%[3]s/%[4]s", mode, id, user, name)).CombinedOutput()
		if err != nil {
			cronBusy = false

			log.Print(out)

			return errors.New("failed to Restore Backup from Backup System")
		}
		deleteDatabaseDump(user, name)
		cronBusy = false

		return nil
	}
	return errors.New("invalid Request")
}

func previousBackupExecuted(t string, frequency string) bool {
	if t == "" {
		log.Print("No timestamp found")
		return false
	}
	switch frequency {
	case "Hourly":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := int(now.Sub(old).Minutes())
		log.Print("Hourly Diff: " + strconv.Itoa(diff))
		if diff > 60 {
			return false
		}
		return true
	case "Daily":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := now.Sub(old).Hours()
		log.Printf("Daily Diff: %f", diff)
		if diff > 24 {
			return false
		}
		return true
	case "Weekly":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := now.Sub(old).Hours()
		log.Printf("Weekly Diff: %f", diff)
		if diff > 168 {
			return false
		}
		return true
	case "Monthly":
		old, _ := time.Parse(time.RFC3339, t)
		now := time.Now().UTC()
		diff := now.Sub(old).Hours()
		log.Printf("Monthly Diff: %f", diff)
		if diff > 648 {
			return false
		}
		return true
	}
	log.Print("No case match")
	return false
}
