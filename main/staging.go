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
	Url := c.Param("url")
	LivesiteUrl := c.Param("livesiteurl")
	logFile, _ := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/staging.log", Name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	logFile.Write([]byte("------------------------------------------------------------------------------\n"))
	logFile.Write([]byte("Starting Staging process\n"))
	logFile.Write([]byte("Time:" + time.Now().String() + "\n"))
	logFile.Write([]byte("Taking ondemad backup of Live site\n"))
	err := takeLocalOndemandBackup(Name, User, true)
	if err != nil {
		LogError(logFile, "Error occured while taking backup", nil, "Staging")
		return c.JSON(http.StatusBadRequest, "Backup process Failed")
	}
	logFile.Write([]byte(fmt.Sprintf("Copying file and folders from %s to %s_Staging\n", Name, Name)))
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("cp -r -p /home/%s/%s /home/%s/%s_Staging", User, Name, User, Name)).CombinedOutput()
	if err != nil {
		LogError(logFile, "Error occured while copying files", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to copy files")
	}
	logFile.Write([]byte("Taking Database Dump\n"))
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", User, Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 {
		LogError(logFile, "Invalid wp-config file configuration", nil, "Staging")
		return errors.New("invalid wp-config file")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -B %s -o /home/%s/%s/private/DatabaseBackup/", dbnameArray[0], User, Name)).CombinedOutput()
	if err != nil {
		deleteDatabaseDump(User, Name)
		LogError(logFile, "Failed to create database dump", out, "Staging")
		return errors.New("database Dump error")
	}
	logFile.Write([]byte("Restoring Database dump to staging database\n"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -d /home/%s/%s/private/DatabaseBackup -o -B %s_Staging", User, Name, Name)).CombinedOutput()
	if err != nil {
		deleteDatabaseDump(User, Name)
		LogError(logFile, "Failed to create staging database", out, "Staging")
		return c.JSON(echo.ErrNotFound.Code, "Failed to create staging database")
	}
	logFile.Write([]byte("Performing database search and replace opteration\n"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s_Staging -u root -p '' -s http://%s -r http://%s -x guid -x user_email", Name, LivesiteUrl, Url)).CombinedOutput()
	if err != nil {
		deleteDatabaseDump(User, Name)
		LogError(logFile, "Failed to create staging database", out, "Staging")
		return errors.New("search and replace operation failed")
	}
	pass, _ := password.Generate(32, 20, 0, false, true)
	logFile.Write([]byte("creating new user and granting access to staging database\n"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"CREATE USER '%s_Staging'@'localhost' IDENTIFIED BY '%s';\"", Name, pass)).CombinedOutput()
	if err != nil {
		deleteDatabaseDump(User, Name)
		LogError(logFile, "Failed to create staging database user", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to create staging user DB")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"GRANT ALL PRIVILEGES ON %s_Staging.* TO '%s_Staging'@'localhost';\"", Name, Name)).CombinedOutput()
	if err != nil {
		deleteDatabaseDump(User, Name)
		LogError(logFile, "Failed to grant privileges to the db", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to grant privileges")
	}
	exec.Command("/bin/bash", "-c", "mysql -e 'FLUSH PRIVILEGES;'").Output()
	deleteDatabaseDump(User, Name)
	logFile.Write([]byte("Replacing wp-config file of staging site with new credentials\n"))
	path := fmt.Sprintf("/home/%s/%s_Staging/public/wp-config.php", User, Name)
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_NAME.*/define( \\'\\'DB_NAME\\'\\\\', \\'\\'%s_Staging\\'\\\\');/' %s", Name, path)).CombinedOutput()
	if err != nil {
		LogError(logFile, "Failed to modify DB_NAME", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_USER.*/define( \\'\\'DB_USER\\'\\\\', \\'\\'%s_Staging\\'\\\\');/' %s", Name, path)).CombinedOutput()
	if err != nil {
		LogError(logFile, "Failed to modify DB_USER", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_PASSWORD.*/define( \\'\\'DB_PASSWORD\\'\\\\', \\'\\'%s\\'\\\\');/' %s", pass, path)).CombinedOutput()

	if err != nil {
		LogError(logFile, "Failed to modify DB_PASSWORD", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to modify wp-config")
	}
	lsws := wpadd{AppName: Name + "_Staging", UserName: User, Url: Url}
	logFile.Write([]byte("Adding site to openlitespeed vhosts\n"))
	err = editLsws(lsws)
	if err != nil {
		LogError(logFile, "Failed to add vhost", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to add vhost")
	}
	logFile.Write([]byte("Adding site to proxy\n"))
	err = addSiteToJSON(lsws, "staging")
	if err != nil {
		LogError(logFile, "Failed to add site", out, "Staging")
		return c.JSON(echo.ErrBadRequest.Code, "Failed to add site to proxy")
	}
	configNuster()
	logFile.Write([]byte("Staging process completed\n"))
	logFile.Close()
	go exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	go exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	return c.JSON(200, "Success")
}

func getDatabaseTables(c echo.Context) error {
	Name := c.Param("name")
	User := c.Param("user")
	var tables []string
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", User, Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 || len(dbnameArray) == 0 {
		return errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_USER | cut -d \\' -f 4", User, Name)).Output()
	dbuser := strings.TrimSuffix(string(db), "\n")
	dbuserArray := strings.Split(dbuser, "\n")
	if len(dbuserArray) > 1 || len(dbuserArray) == 0 {

		return errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_PASSWORD | cut -d \\' -f 4", User, Name)).Output()
	dbpassword := strings.TrimSuffix(string(db), "\n")
	dbpasswordArray := strings.Split(dbpassword, "\n")
	if len(dbpasswordArray) > 1 || len(dbpasswordArray) == 0 {

		return errors.New("invalid wp-config file")
	}
	mysql, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s", dbuserArray[0], dbpasswordArray[0], dbnameArray[0]))
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

func syncChanges(c echo.Context) error {
	var sync SyncChanges
	if err := c.Bind(&sync); err != nil {
		return c.JSON(400, err.Error())
	}
	var live string
	if sync.From.Type == "live" {
		live = sync.From.Name
	} else {
		live = sync.To.Name
	}
	logFile, _ := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/staging.log", live), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	logFile.Write([]byte("------------------------------------------------------------------------------\n"))
	logFile.Write([]byte("Starting Sync Process\n"))
	logFile.Write([]byte("Time:" + time.Now().String() + "\n"))
	logFile.Write([]byte(fmt.Sprintf("Taking ondemad backup of %s site\n", sync.To.Name)))
	//First take backup of toSite
	err := takeLocalOndemandBackup(sync.To.Name, sync.To.User, false)
	latestBackupByte, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/ondemand ;  kopia snapshot list /home/%s/%s/ -l | tail -1 | awk '{print $4}'", sync.To.User, sync.To.Name)).Output()
	latestBackup := string(latestBackupByte)
	if err != nil {
		return c.JSON(404, "Cannot Take backup, Sync process Stoped")
	}
	for _, syncType := range sync.Type {
		if syncType == "files" {
			err := syncCopyFiles(sync, logFile)
			if err != nil {
				restoreBackup(sync.To.Name, sync.To.User, latestBackup, "webapp", "ondemand")
				return c.JSON(404, "Failed to copy files")
			}
		} else if syncType == "db" {
			shouldRestore := false
			err := syncCopyDb(sync, logFile, &shouldRestore)
			if err != nil {
				deleteDatabaseDump(sync.From.User, sync.From.Name)
				if shouldRestore {
					restoreBackup(sync.To.Name, sync.To.User, latestBackup, "db", "ondemand")
				}
				return c.JSON(404, "Failed to sync DB")
			}
		}
	}
	logFile.Write([]byte("Sync process successful\n"))
	return nil
}

func syncCopyFiles(sync SyncChanges, logFile *os.File) error {
	source := "/home/" + sync.From.User + "/" + sync.From.Name
	dest := "/home/" + sync.To.User + "/" + sync.To.Name
	logFile.Write([]byte("Started File copying process\n"))
	//get db name,user,password of toSite before rsync
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", sync.To.User, sync.To.Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 || len(dbnameArray) == 0 {
		return errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_USER | cut -d \\' -f 4", sync.To.User, sync.To.Name)).Output()
	dbuser := strings.TrimSuffix(string(db), "\n")
	dbuserArray := strings.Split(dbuser, "\n")
	if len(dbuserArray) > 1 || len(dbuserArray) == 0 {

		return errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_PASSWORD | cut -d \\' -f 4", sync.To.User, sync.To.Name)).Output()
	dbpassword := strings.TrimSuffix(string(db), "\n")
	dbpasswordArray := strings.Split(dbpassword, "\n")
	if len(dbpasswordArray) > 1 || len(dbpasswordArray) == 0 {

		return errors.New("invalid wp-config file")
	}
	//copy files
	logFile.Write([]byte(fmt.Sprintf("Copying files from /%s/%s to /%s/%s \n", sync.From.User, sync.From.Name, sync.To.User, sync.To.Name)))
	logFile.Write([]byte(fmt.Sprintf("Performing %s operation", sync.CopyMethod)))
	if sync.CopyMethod == "overwrite" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("rsync -ar --delete %s/ %s", source, dest)).CombinedOutput()
		if err != nil {
			LogError(logFile, "Error copying files", out, "Sync")
			return errors.New(string(out))
		}
	} else {
		var exclude string
		for _, file := range sync.Exclude.Files {

			exclude = exclude + fmt.Sprintf("'%s',", file)

		}
		for _, folder := range sync.Exclude.Folders {

			exclude = exclude + fmt.Sprintf("'%s',", folder)

		}
		if len(sync.Exclude.Files) == 0 && len(sync.Exclude.Folders) == 0 {
			out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("rsync -ar %s/ %s", source, dest)).CombinedOutput()
			if err != nil {
				LogError(logFile, "Error copying files", out, "Sync")
				return errors.New(string(out))
			}
		}
		if sync.DeleteDestFiles {
			out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("rsync -ar --delete --exclude={%s} %s/ %s", exclude, source, dest)).CombinedOutput()
			if err != nil {
				LogError(logFile, "Error copying files", out, "Sync")
				return errors.New(string(out))
			}
		} else {
			out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("rsync -ar --exclude={%s} %s/ %s", exclude, source, dest)).CombinedOutput()
			if err != nil {
				LogError(logFile, "Error copying files", out, "Sync")
				return errors.New(string(out))
			}
		}
	}
	//replace wp-config file with old db credientials
	path := fmt.Sprintf("/home/%s/%s/public/wp-config.php", sync.To.User, sync.To.Name)
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_NAME.*/define( \\'\\'DB_NAME\\'\\\\', \\'\\'%s\\'\\\\');/' %s", dbname, path)).CombinedOutput()
	if err != nil {
		LogError(logFile, "Failed to modify DB_NAME", out, "Staging")
		return errors.New("failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_USER.*/define( \\'\\'DB_USER\\'\\\\', \\'\\'%s\\'\\\\');/' %s", dbuser, path)).CombinedOutput()
	if err != nil {
		LogError(logFile, "Failed to modify DB_USER", out, "Staging")
		return errors.New("failed to modify wp-config")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/.*DB_PASSWORD.*/define( \\'\\'DB_PASSWORD\\'\\\\', \\'\\'%s\\'\\\\');/' %s", dbpassword, path)).CombinedOutput()

	if err != nil {
		LogError(logFile, "Failed to modify DB_PASSWORD", out, "Staging")
		return errors.New("failed to modify wp-config")
	}
	return nil
}

func syncCopyDb(sync SyncChanges, logFile *os.File, shouldRestore *bool) error {

	logFile.Write([]byte("Taking Database Dump\n"))
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", sync.From.User, sync.From.Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 {
		LogError(logFile, "Invalid wp-config file configuration", nil, "Sync")
		return errors.New("invalid wp-config file")
	}
	if sync.AllSelected || sync.DbType == "full" {
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -B %s -o /home/%s/%s/private/DatabaseBackup/", dbnameArray[0], sync.From.User, sync.From.Name)).CombinedOutput()
		if err != nil {
			LogError(logFile, "Failed to create database dump", out, "Sync")
			return errors.New("database Dump error")
		}
	} else {
		logFile.Write([]byte("Following tables are being Dumped\n"))
		var dumpTable string
		dumpLength := len(sync.Tables)
		for i, table := range sync.Tables {
			if i == (dumpLength - 1) {
				dumpTable = dumpTable + table
				logFile.Write([]byte(table + "\n"))
				break
			}
			dumpTable = dumpTable + table + ","
			logFile.Write([]byte(table + "\t"))
		}
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mydumper -B %s -o /home/%s/%s/private/DatabaseBackup/ -T %s", dbnameArray[0], sync.From.User, sync.From.Name, dumpTable)).CombinedOutput()
		if err != nil {
			LogError(logFile, "Failed to create database dump", out, "Sync")
			return errors.New("database Dump error")
		}
	}

	logFile.Write([]byte(fmt.Sprintf("Collecting DB information of %s site \n", sync.To.Name)))
	toDb, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", sync.To.User, sync.To.Name)).Output()
	toDbname := strings.TrimSuffix(string(toDb), "\n")
	toDbnameArray := strings.Split(toDbname, "\n")
	if len(toDbnameArray) > 1 {
		LogError(logFile, fmt.Sprintf("Invalid wp-config file configuration on %s site", sync.To.Name), nil, "Sync")
		return errors.New("invalid wp-config file")
	}
	*shouldRestore = true
	logFile.Write([]byte(fmt.Sprintf("Copying %s site Database to %s site database\n", sync.From.Name, sync.To.Name)))

	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("myloader -d /home/%s/%s/private/DatabaseBackup -o -B %s", sync.From.User, sync.From.Name, toDbnameArray[0])).CombinedOutput()
	if err != nil {
		LogError(logFile, "Failed to copy database", out, "Sync")
		return errors.New("failed to copy database")
	}

	logFile.Write([]byte("Performing database search and replace opteration\n"))
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s -u root -p '' -s http://%s -r http://%s -x guid -x user_email", toDbname, sync.From.Url, sync.To.Url)).CombinedOutput()
	if err != nil {
		LogError(logFile, "Failed to Search and Replace url in database", out, "Sync")
		return errors.New("search and replace operation failed")
	}
	deleteDatabaseDump(sync.From.User, sync.From.Name)
	return nil
}

func deleteDatabaseDump(user string, name string) {
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s/private/DatabaseBackup", user, name)).Output()
}

func deleteStagingSite(c echo.Context) error {
	name := c.Param("name")
	user := c.Param("user")
	err := deleteStagingSiteInternal(name, user)
	if err != nil {
		c.JSON(404, err)
	}
	return c.JSON(200, "success")
}

func deleteStagingSiteInternal(name string, user string) error {
	// logFile, _ := os.OpenFile(fmt.Sprintf("/var/log/hosting/%s/staging.log", name), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	// logFile.Write([]byte("------------------------------------------------------------------------------\n"))
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", user, name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 || len(dbnameArray) == 0 {
		return errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_USER | cut -d \\' -f 4", user, name)).Output()
	dbuser := strings.TrimSuffix(string(db), "\n")
	dbuserArray := strings.Split(dbuser, "\n")
	if len(dbuserArray) > 1 || len(dbuserArray) == 0 {
		return errors.New("invalid wp-config file")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s", user, name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /usr/local/lsws/conf/vhosts/%s.*", name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"DROP DATABASE %s;\"", name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"DROP USER '%s'@'localhost';\"", name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/ondemand --password=kopia ; kopia snapshot delete --all-snapshots-for-source /home/%s/%s --delete", user, name)).Output()
	deleteSiteFromJSON(name)
	go exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	go exec.Command("/bin/bash", "-c", "service lsws restart").Output()

	configNuster()

	// exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/%s\\/%s/d' /etc/incron.d/sites.txt", user, name)).Output()
	go exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	return nil
}

func LogError(logFile *os.File, errorStage string, output []byte, process string) {
	logFile.Write([]byte(errorStage + "/n"))
	logFile.Write(output)
	logFile.Write([]byte(fmt.Sprintf("%s process failed\n", process)))
	logFile.Close()
}
