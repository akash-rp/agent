package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"

	"github.com/labstack/echo/v4"
)

func siteCloneRequest(c echo.Context) error {
	data := new(Clone)
	c.Bind(&data)
	dbCred, err := siteClone(*data)
	if err != nil {
		return AbortWithErrorMessage(c, err.Error())
	}
	dbc := make(map[string]string)
	dbc["name"] = dbCred.Name
	dbc["user"] = dbCred.User
	return c.JSON(200, dbc)
}

func siteClone(data Clone) (db, error) {

	if data.Original.Name == "" || data.Original.Domain == "" || data.Original.User == "" || data.Clone.Domain.Url == "" || data.Clone.Name == "" || data.Clone.User == "" {
		return db{}, errors.New("invalid Fields")
	}
	//check if wordpress exists
	if _, err := os.Stat(fmt.Sprintf("/home/%s/%s/public/wp-config.php", data.Original.User, data.Original.Name)); err != nil {
		return db{}, errors.New("invalid app path")
	}

	//root password required to my loader
	rootPass, err := getMariadbRootPass()
	if err != nil {
		return db{}, errors.New("iootPass not found")
	}
	//dump database to original site private folder
	if err = mydumper(data.Original.User, data.Original.Name, ""); err != nil {
		return db{}, errors.New("mydumper error")
	}
	//create new directory for clone site in lsws
	_, err = linuxCommand(fmt.Sprintf("mkdir /usr/local/lsws/conf/vhosts/%s.d", data.Clone.Name))
	if err != nil {
		return db{}, errors.New("cannot create folder is lsws")
	}

	//copy lsws config of original site to clone site
	_, err = linuxCommand(fmt.Sprintf("cp -r /usr/local/lsws/conf/vhosts/%s.d/* /usr/local/lsws/conf/vhosts/%s.d/", data.Original.Name, data.Clone.Name))
	if err != nil {
		return db{}, errors.New("cannot cp config files")
	}

	//remove domains files in clone site conf
	_, err = linuxCommand(fmt.Sprintf("rm /usr/local/lsws/conf/vhosts/%s.d/domain/*", data.Clone.Name))
	if err != nil {
		return db{}, errors.New("cannot remove domain files")
	}

	//write clone conf file to point to its directory
	if err = os.WriteFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.conf", data.Clone.Name), []byte(fmt.Sprintf("include /usr/local/lsws/conf/vhosts/%s.d/domain/*.conf", data.Clone.Name)), 0640); err != nil {
		return db{}, errors.New("cannot write domain file")
	}

	//write domain file to clone site
	if err = addDomainConf(data.Clone.Domain, data.Clone.Name); err != nil {
		return db{}, errors.New("failed to add domain conf file")
	}

	//replace site name and user to clone site in main.conf
	if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[2]s.d/main.conf", data.Original.Name, data.Clone.Name)); err != nil {
		return db{}, errors.New("failed to replace site name")
	}
	if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[3]s.d/main.conf", data.Original.User, data.Clone.User, data.Clone.Name)); err != nil {
		return db{}, errors.New("failed to replace site name")
	}

	//replace site name and user to clone site in extphp
	if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[2]s.d/modules/extphp.conf", data.Original.Name, data.Clone.Name)); err != nil {
		return db{}, errors.New("failed to replace site name in extphp")
	}
	if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[3]s.d/modules/extphp.conf", data.Original.User, data.Clone.User, data.Clone.Name)); err != nil {
		return db{}, errors.New("failed to replace site name in extphp")
	}

	//replace site name and user to clone site in context
	if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[2]s.d/modules/context.conf", data.Original.Name, data.Clone.Name)); err != nil {
		return db{}, errors.New("failed to replace site name in context")
	}
	if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[3]s.d/modules/context.conf", data.Original.User, data.Clone.User, data.Clone.Name)); err != nil {
		return db{}, errors.New("failed to replace site name in context")
	}

	//check if modsecurity file exists and replace site name in that file
	if _, err := os.Stat(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/modsecurity.conf", data.Original.Name)); err == nil {
		if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[2]s.d/modules/modsecurity.conf", data.Original.Name, data.Clone.Name)); err != nil {
			return db{}, errors.New("failed to replace site name in modsecurity")
		}
	}

	//check if modsecurity main.conf file exists and replace site name in that file
	if _, err := os.Stat(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/modsecurity.d/main.conf", data.Original.Name)); err == nil {
		if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[2]s.d/modules/modsecurity.d/main.conf", data.Original.Name, data.Clone.Name)); err != nil {
			return db{}, errors.New("failed to replace site name in modsecurity")
		}
	}

	//check if siteauth file exists and replace site name in that file
	if _, err := os.Stat(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/siteauth.conf", data.Original.Name)); err == nil {
		if _, err := linuxCommand(fmt.Sprintf("sed -i 's/%[1]s/%[2]s/g' /usr/local/lsws/conf/vhosts/%[2]s.d/modules/siteauth.conf", data.Original.Name, data.Clone.Name)); err != nil {
			return db{}, errors.New("failed to replace site name in modsecurity")
		}
	}

	//create empty database for clone site
	dbCred, err := createDatabase(data.Clone.Name)
	if err != nil {
		return db{}, errors.New(err.Error())
	}

	//dump original site database to new clone site
	out, err := linuxCommand(fmt.Sprintf("myloader -u root -p %s -d /home/%s/%s/private/DatabaseBackup -o -B %s", rootPass, data.Original.User, data.Original.Name, dbCred.Name))
	if err != nil {
		return db{}, errors.New(string(out))
	}

	//rewrite domain url to new clone url
	out, err = linuxCommand(fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s -u root -p %s -s http://%s -r http://%s -x guid -x user_email", dbCred.Name, rootPass, data.Original.Domain, data.Clone.Domain.Url))
	if err != nil {
		return db{}, errors.New(string(out))
	}

	//delete database dump in original site private folder
	linuxCommand(fmt.Sprintf("rm -rf /home/%s/%s/private/DatabaseBackup", data.Original.User, data.Original.Name))

	// check if user exists or not. If not then create a user with home directory
	_, err = user.Lookup(data.Clone.User)
	if err != nil {
		linuxCommand(fmt.Sprintf("useradd --shell /bin/bash --create-home %s", data.Clone.User))
	}

	//copy wordpress files from original site to clone site
	out, err = linuxCommand(fmt.Sprintf("cp -r -p /home/%s/%s /home/%s/%s", data.Original.User, data.Original.Name, data.Clone.User, data.Clone.Name))
	if err != nil {
		return db{}, errors.New(string(out))
	}

	//set permission for cloned folder
	if err := fixFilePermission(data.Clone.Name, data.Clone.User); err != nil {
		return db{}, err
	}

	//set db crediantls in clone site wp-config.php
	out, err = linuxCommand(fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli config set DB_NAME %s --path=/home/%s/%s/public/", data.Clone.User, dbCred.Name, data.Clone.User, data.Clone.Name))
	if err != nil {
		return db{}, errors.New(string(out))
	}
	out, err = linuxCommand(fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli config set DB_USER %s --path=/home/%s/%s/public/", data.Clone.User, dbCred.User, data.Clone.User, data.Clone.Name))
	if err != nil {
		return db{}, errors.New(string(out))
	}
	out, err = linuxCommand(fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli config set DB_PASSWORD %s --path=/home/%s/%s/public/", data.Clone.User, dbCred.Password, data.Clone.User, data.Clone.Name))
	if err != nil {
		return db{}, errors.New(string(out))
	}

	//add site to json
	addSiteToJSON(data.Clone.Name, data.Clone.User, data.Clone.Domain.Url, "live")
	return dbCred, nil
}
