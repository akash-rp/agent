package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"
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
		_ = cronInt.RemoveByTag(name)
		latest := getLatest(backup.Frequency, backup.Retention)
		out, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/automatic/automatic.config policy set /home/%s/%s --keep-latest %d --keep-hourly 0 --keep-daily 0 --keep-weekly 0 --keep-monthly 0 --keep-annual 0 ", user, name, latest)).CombinedOutput()
		log.Println(string(out))
		found := false
		for i, site := range obj.Sites {
			if site.Name == name {
				lastBackup = site.LocalBackup.LastRun
				err := addCronJob(*backup, name, user, lastBackup)
				if err != nil {
					return errors.New("error adding cron job")
				}
				obj.Sites[i].LocalBackup = *backup
				if lastBackup == "" {
					obj.Sites[i].LocalBackup.LastRun = time.Now().UTC().Format(time.RFC3339)
				} else {
					obj.Sites[i].LocalBackup.LastRun = lastBackup
				}
				found = true
			}
		}
		if !found {
			log.Print(fmt.Sprintf("Site not found %s %+v", name, backup))
			return errors.New("eror")
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

func takeBackup(name string, user string, msg string, backupLocation string, provider string, bucket string) {
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

	err = mydumper(user, name, "")
	if err != nil {
		f.Write([]byte(err.Error()))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()
	var out []byte
	if backupLocation == "local" {
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
	} else {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/remote/%s.config snapshot create /home/%s/%s --tags=type:automatic", provider+"-"+bucket, user, name)).CombinedOutput()
		if err != nil {
			f.Write([]byte("Cannot create backup"))
			f.Write([]byte(string(out)))
		}
		for i, site := range obj.Sites {
			if site.Name == name {
				for ri, remote := range site.RemoteBackup {
					if remote.Provider == provider && remote.Bucket == bucket {
						obj.Sites[i].RemoteBackup[ri].LastRun = time.Now().UTC().Format(time.RFC3339)
					}
				}
			}
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
		cron := backup.getCronExpression()
		if cron == "" {
			return errors.New("invalid backup frequency")
		}

		_, err = cronInt.Cron(cron).Tag(name).Do(func() {
			takeBackup(name, user, "Started by cron Do function", "local", "", "")
		})
		if err != nil {
			log.Print(err)
		}

		log.Print("Next is Prevoious function")

		if !previousBackupExecuted(lastBackup, backup.Frequency) {
			takeBackup(name, user, "Started Due to no last run", "local", "", "")
		}
		return nil

	}

	if err != nil {
		return err
	}

	return nil
}

func addRemoteCronJob(backup RemoteBackup, name string, user string, lastBackup string) error {
	log.Print(backup)
	var err error
	if (backup != RemoteBackup{}) {
		cron := backup.getCronExpression()
		if cron == "" {
			return errors.New("invalid backup frequency")
		}
		_, err = cronInt.Cron(cron).Tag(name + "-" + backup.Provider + "-" + backup.Bucket).Do(func() {
			takeBackup(name, user, "Started by cron Do function", "remote", backup.Provider, backup.Bucket)
		})
		if err != nil {
			log.Print(err)
		}

		log.Print("Next is Previous function")

		if !previousBackupExecuted(lastBackup, backup.Frequency) {
			takeBackup(name, user, "Started Due to no last run", "remote", backup.Provider, backup.Bucket)
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

func getLatest(freq string, retention BackupRetention) int {
	latest := 0
	switch freq {
	case "Hourly":
		switch retention.Type {
		case "Day":
			log.Print("Entered Day case")
			latest = 24 * retention.Time
			log.Print("Latest: " + strconv.Itoa(latest))
		case "Week":
			latest = 24 * 7 * retention.Time
		case "Month":
			latest = 24 * 28 * retention.Time
		}
	case "Daily":
		switch retention.Type {
		case "Day":
			latest = 1 * retention.Time
		case "Week":
			latest = 7 * retention.Time
		case "Month":
			latest = 28 * retention.Time
		}
	case "Weekly":
		switch retention.Type {
		case "Week":
			latest = 1 * retention.Time
		case "Month":
			latest = 4 * retention.Time
		}
	case "Monthly":
		switch retention.Type {
		case "Month":
			latest = 1 * retention.Time
		}
	}
	return latest
}

func ondemadBackup(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	location := c.Param("type")
	data := new(struct {
		Tag     string `json:"tag"`
		Storage string `json:"storage"`
	})
	c.Bind(&data)
	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true
	var err error
	f, err := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/backup.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Print(err)
	}
	f.Write([]byte("\n--------------------------------------------------------------------------------------\n"))
	f.Write([]byte("ONDEMAND Backup Process started\n"))
	f.Write([]byte("Time:" + time.Now().String() + "\n"))
	err = mydumper(user, name, "")
	if err != nil {
		f.Write([]byte("Failed to create database backup"))
		f.Write([]byte(err.Error()))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return c.JSON(http.StatusBadRequest, "database Dump error")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()
	var out []byte
	if location == "local" {
		out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/ondemand/ondemand.config snapshot create /home/%s/%s --description '%s'", user, name, data.Tag)).CombinedOutput()
	} else if location == "remote" {
		out, err = linuxCommand(fmt.Sprintf("kopia --config-file=/var/Backup/config/remote/%s.config snapshot create /home/%s/%s --description '%s' --tags=type:ondemand", data.Storage, user, name, data.Tag))
	}

	if err != nil {
		f.Write([]byte("Cannot create backup"))
		f.Write([]byte(string(out)))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return c.JSON(http.StatusBadRequest, "cannot create backup")

	} else {
		f.Write([]byte("Backup Process Completed\n"))
		f.Close()
		cronBusy = false
	}

	deleteDatabaseDump(user, name)
	if location == "local" {

		list, _ := LocalBackupsList(name, user)
		return c.JSON(http.StatusOK, list)
	} else if location == "remote" {
		list, _ := RemoteBackupsList(name, user, data.Storage)
		return c.JSON(200, list)
	}
	return c.NoContent(400)
}

func takeSystemBackup(name string, user string, Description string) error {
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
	err = mydumper(user, name, "")
	if err != nil {
		f.Write([]byte("Failed to create database backup"))
		f.Write([]byte(err.Error()))
		f.Write([]byte("Backup Process Failed"))
		f.Close()
		cronBusy = false
		return errors.New("database Dump error")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/private/DatabaseBackup/metadata", user, name)).Output()

	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/system/system.config snapshot create /home/%s/%s --description='%s' ", user, name, Description)).CombinedOutput()

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
		return AbortWithErrorMessage(c, err.Error())
	}
	return c.JSON(200, list)
}

func LocalBackupsList(name string, user string) (LocalBackupList, error) {

	ondemand, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/ondemand/ondemand.config snapshot list /home/%s/%s --json", user, name)).CombinedOutput()
	if err != nil {

		return LocalBackupList{}, errors.New("cannot list backups")
	}

	automatic, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/automatic/automatic.config snapshot list /home/%s/%s --json", user, name)).CombinedOutput()
	if err != nil {

		return LocalBackupList{}, errors.New("cannot list backups")
	}

	system, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/system/system.config snapshot list /home/%s/%s --json", user, name)).CombinedOutput()
	if err != nil {

		return LocalBackupList{}, errors.New("cannot list backups")
	}
	list := new(LocalBackupList)
	json.Unmarshal(ondemand, &list.Ondemand)
	json.Unmarshal(automatic, &list.Automatic)
	json.Unmarshal(system, &list.System)

	return *list, nil

}

func getRemoteBackupsList(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	storage := c.Param("storage")
	list, err := RemoteBackupsList(name, user, storage)
	if err != nil {
		return AbortWithErrorMessage(c, err.Error())
	}
	return c.JSON(200, list)
}

func RemoteBackupsList(name string, user string, storage string) (RemoteBackupList, error) {

	automatic, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/remote/%s.config snapshot list /home/%s/%s --json  --tags=type:automatic", storage, user, name)).CombinedOutput()
	if err != nil {

		return RemoteBackupList{}, errors.New("cannot list backups")
	}

	ondemand, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/remote/%s.config snapshot list /home/%s/%s --json --tags=type:ondemand", storage, user, name)).CombinedOutput()
	if err != nil {

		return RemoteBackupList{}, errors.New("cannot list backups")
	}

	list := new(RemoteBackupList)
	json.Unmarshal(ondemand, &list.Ondemand)
	json.Unmarshal(automatic, &list.Automatic)

	return *list, nil

}

func restoreBackupFromPanel(c echo.Context) error {
	type restoreBackupData struct {
		Name    string `json:"name"`
		User    string `json:"user"`
		Restore struct {
			Mode     string `json:"mode"`
			Id       string `json:"id"`
			Type     string `json:"type"`
			Provider string `json:"provider"`
			Bucket   string `json:"bucket"`
		}
	}

	backup := new(restoreBackupData)
	c.Bind(&backup)
	err := restoreBackup(backup.Name, backup.User, backup.Restore.Id, backup.Restore.Type, backup.Restore.Mode, backup.Restore.Provider, backup.Restore.Bucket)
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}
	return c.JSON(200, "success")
}

func restoreBackup(name string, user string, id string, restoreType string, mode string, provider string, bucket string) error {

	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true
	rootPass, err := getMariadbRootPass()
	if err != nil {
		return errors.New("root password not found")
	}
	var conf string

	if mode == "remote" {
		conf = "remote/" + provider + "-" + bucket
	} else {
		conf = mode + "/" + mode
	}

	if restoreType == "both" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/%[1]s.config restore %[2]s /home/%[3]s/%[4]s", conf, id, user, name)).CombinedOutput()
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
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/%[1]s.config restore %[2]s/DatabaseBackup /home/%[3]s/%[4]s/private/DatabaseBackup", conf, id, user, name)).CombinedOutput()
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
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/%[1]s.config restore %[2]s /home/%[3]s/%[4]s", conf, id, user, name)).CombinedOutput()
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

func mydumper(User string, Name string, Tables string) error {
	rootPass, err := getMariadbRootPass()
	if err != nil {
		return errors.New(err.Error())
	}
	dbname, _, _, err := getDbcredentials(User, Name)
	if err != nil {
		return errors.New(err.Error())
	}
	if Tables == "" {

		dbout, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -u root -p %s -B %s -o /home/%s/%s/private/DatabaseBackup/", rootPass, dbname, User, Name)).CombinedOutput()
		if err != nil {
			return errors.New(string(dbout))
		}
	} else {
		dbout, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -u root -p %s -B %s -o /home/%s/%s/private/DatabaseBackup/ -T %s", rootPass, dbname, User, Name, Tables)).CombinedOutput()
		if err != nil {
			return errors.New(string(dbout))
		}
	}
	return nil
}

func BackupDownload(c echo.Context) error {
	mode := c.Param("mode")
	id := c.Param("id")
	out, err := linuxCommand(fmt.Sprintf("kopia --config-file=/var/Backup/config/%[1]s/%[1]s.config snapshot restore %[2]s /usr/Hosting/tmp/%[2]s ; cd /usr/Hosting/tmp/ ; zip -r %[2]s.zip %[2]s/ ; cd", mode, id))
	if err != nil {
		log.Print(string(out))
		return c.NoContent(404)
	}
	file := fmt.Sprintf("/usr/Hosting/tmp/%s.zip", id)

	err = c.File(file)
	if err != nil {
		log.Print(err.Error())
	}
	defer linuxCommand(fmt.Sprintf("rm -rf /usr/Hosting/tmp/%s*", id))
	return c.NoContent(200)
}

func AddRemoteBackupCredentials(c echo.Context) error {
	name := c.Param("name")
	data := new(struct {
		Provider  string `json:"provider"`
		AccessKey string `json:"accessKey"`
		SecretKey string `json:"secretKey"`
		Bucket    string `json:"bucket"`
		Id        string `json:"id"`
		Endpoint  string `json:"endpoint"`
	})
	c.Bind(&data)
	switch data.Provider {
	case "backblaze":
		_, err := linuxCommand(fmt.Sprintf("kopia repository create b2 --config-file=/var/Backup/config/remote/%s.config --bucket=%s --key-id=%s --key=%s --password=%s", data.Provider+"-"+data.Bucket, data.Bucket, data.AccessKey, data.SecretKey, data.Id))
		if err != nil {
			_, err := linuxCommand(fmt.Sprintf("kopia repository connect b2 --config-file=/var/Backup/config/remote/%s.config --bucket=%s --key-id=%s --key=%s --password=%s", data.Provider+"-"+data.Bucket, data.Bucket, data.AccessKey, data.SecretKey, data.Id))
			if err != nil {
				json := new(errJson)
				json.Errors = append(json.Errors, struct {
					Field   string "json:\"field\""
					Message string "json:\"message\""
				}{Field: "bucket", Message: "Bucket is neither empty nor backup system belongs to this user"})
				return c.JSON(400, json)
			}

		}

	case "wasabi", "aws":
		out, err := linuxCommand(fmt.Sprintf("kopia repository create s3 --config-file=/var/Backup/config/remote/%s.config --bucket=%s --access-key=%s --secret-access-key=%s --endpoint=%s --password=%s", data.Provider+"-"+data.Bucket, data.Bucket, data.AccessKey, data.SecretKey, data.Endpoint, data.Id))
		if err != nil {
			log.Print(string(out))
			out, err := linuxCommand(fmt.Sprintf("kopia repository connect s3 --config-file=/var/Backup/config/remote/%s.config --bucket=%s --access-key=%s --secret-access-key=%s --endpoint=%s --password=%s", data.Provider+"-"+data.Bucket, data.Bucket, data.AccessKey, data.SecretKey, data.Endpoint, data.Id))
			if err != nil {
				log.Print(string(out))
				json := new(errJson)
				json.Errors = append(json.Errors, struct {
					Field   string "json:\"field\""
					Message string "json:\"message\""
				}{Field: "bucket", Message: "Bucket is neither empty nor backup system belongs to this user"})
				return c.JSON(400, json)
			}

		}
	default:
		json := new(errJson)
		json.Errors = append(json.Errors, struct {
			Field   string "json:\"field\""
			Message string "json:\"message\""
		}{Field: "provider", Message: "Invalid provider"})
		return c.JSON(400, json)
	}

	backup := RemoteBackup{
		Provider: data.Provider,
		Bucket:   data.Bucket,
		Backup: Backup{
			Automatic: false,
			Frequency: "Hourly",
			LastRun:   "",
			Retention: BackupRetention{Type: "Day", Time: 7},
			Time:      BackupTime{Hour: "00", Minute: "00", MonthDay: "00", WeekDay: "Sunday"},
		},
	}

	for i, site := range obj.Sites {
		if site.Name == name {
			remoteExists := false
			for _, remote := range site.RemoteBackup {
				if remote.Provider == data.Provider && remote.Bucket == data.Bucket {
					remoteExists = true
				}
			}
			if remoteExists {
				json := new(errJson)
				json.Errors = append(json.Errors, struct {
					Field   string "json:\"field\""
					Message string "json:\"message\""
				}{Field: "bucket", Message: "Bucket exists"})
				return c.JSON(400, json)
			}
			obj.Sites[i].RemoteBackup = append(obj.Sites[i].RemoteBackup, backup)
			break
		}
	}
	SaveJSONFile()
	return c.NoContent(200)
}

func updateRemoteBackup(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	backup := new(RemoteBackup)
	c.Bind(&backup)
	lastBackup := ""
	if backup.Automatic {
		cronInt.RemoveByTag(name + "-" + backup.Provider + "-" + backup.Bucket)
		latest := getLatest(backup.Frequency, backup.Retention)
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/remote/%s.config policy set /home/%s/%s --keep-latest %d --keep-hourly 0 --keep-daily 0 --keep-weekly 0 --keep-monthly 0 --keep-annual 0 ", backup.Provider+"-"+backup.Bucket, user, name, latest)).CombinedOutput()
		if err != nil {
			log.Print(string(out))
			return c.NoContent(400)
		}
		log.Println(string(out))

		for i, site := range obj.Sites {
			if site.Name == name {
				for remoteIndex, remote := range site.RemoteBackup {
					if remote.Provider == backup.Provider && remote.Bucket == backup.Bucket {
						log.Print("Found remote provider in site config")
						lastBackup = remote.LastRun
						err := addRemoteCronJob(*backup, name, user, lastBackup)
						if err != nil {
							return AbortWithErrorMessage(c, err.Error())
						}
						log.Print("next to save backup config")
						obj.Sites[i].RemoteBackup[remoteIndex] = *backup
						log.Printf("%+v", obj)
						if lastBackup == "" {
							obj.Sites[i].RemoteBackup[remoteIndex].LastRun = time.Now().UTC().Format(time.RFC3339)
						} else {
							obj.Sites[i].RemoteBackup[remoteIndex].LastRun = lastBackup
						}
						break
					}

				}
				break
			}

		}

		SaveJSONFile()
		return c.NoContent(200)

	} else {
		cronInt.RemoveByTag(name + "-" + backup.Provider + "-" + backup.Bucket)
		for i, site := range obj.Sites {
			if site.Name == name {
				for ri, remote := range site.RemoteBackup {
					if remote.Provider == backup.Provider && remote.Bucket == backup.Bucket {
						obj.Sites[i].RemoteBackup[ri].Automatic = false
						break
					}
				}
				break
			}
		}
		SaveJSONFile()
		return c.NoContent(200)
	}
}

func (backup *Backup) getCronExpression() string {
	cron := ""
	if backup.Automatic {
		switch backup.Frequency {
		case "Hourly":
			cron = fmt.Sprintf("%s * * * *", backup.Time.Minute)
		case "Daily":
			cron = fmt.Sprintf("%s %s * * *", backup.Time.Minute, backup.Time.Hour)
		case "Weekly":
			switch backup.Time.WeekDay {
			case "Sunday":
				cron = fmt.Sprintf("%s %s * * 0", backup.Time.Minute, backup.Time.Hour)
			case "Monday":
				cron = fmt.Sprintf("%s %s * * 1", backup.Time.Minute, backup.Time.Hour)
			case "Tuesday":
				cron = fmt.Sprintf("%s %s * * 2", backup.Time.Minute, backup.Time.Hour)
			case "Wednesday":
				cron = fmt.Sprintf("%s %s * * 3", backup.Time.Minute, backup.Time.Hour)
			case "Thursday":
				cron = fmt.Sprintf("%s %s * * 4", backup.Time.Minute, backup.Time.Hour)
			case "Friday":
				cron = fmt.Sprintf("%s %s * * 5", backup.Time.Minute, backup.Time.Hour)
			case "Saturday":
				cron = fmt.Sprintf("%s %s * * 6", backup.Time.Minute, backup.Time.Hour)
			}
		case "Monthly":
			day, _ := strconv.Atoi(backup.Time.MonthDay)
			cron = fmt.Sprintf("%s %s %s * *", backup.Time.Minute, backup.Time.Hour, day)
		}
	}
	return cron
}
