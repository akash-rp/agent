package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sethvargo/go-password/password"
)

func wpAdd(c echo.Context) error {
	// Bind received post request body to a struct
	wp := new(wpadd)
	c.Bind(&wp)
	f, err := os.OpenFile("/usr/Hosting/error.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return c.JSON(404, err)
	}

	// Check if all fields are defind
	if wp.AppName == "" || wp.Url == "" || wp.UserName == "" || wp.Title == "" || wp.AdminEmail == "" || wp.AdminPassword == "" || wp.AdminUser == "" {
		result := &errcode{
			Code:    101,
			Message: "Required fields are not defined",
		}
		return c.JSON(http.StatusBadRequest, result)
	}

	// check if user exists or not. If not then create a user with home directory
	_, err = user.Lookup(wp.UserName)
	if err != nil {
		exec.Command("/bin/bash", "-c", fmt.Sprintf("useradd --shell /bin/bash --create-home %s", wp.UserName)).Output()
	}

	// Assign path of home directory to a variable
	path := fmt.Sprintf("/home/%s/", wp.UserName)

	// check if path exists. If not then create a directory
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir %s", path)).Output()
	}

	// Check for appName if already exists or not. If exists send error message
	lsByte, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("ls %s", path)).Output()
	lsStirng := string(lsByte)
	lsSlice := strings.Split(lsStirng, "\n")

	for _, ls := range lsSlice {
		if ls == wp.AppName {
			result := &errcode{
				Code:    102,
				Message: "App Name exists",
			}
			return c.JSON(http.StatusBadRequest, result)
		}
	}

	// Create random number to concate with appName for prevention of Duplicity. Create rand password for DB password and assign them to DB struct
	randInt, _ := password.Generate(5, 5, 0, false, true)
	pass, _ := password.Generate(32, 20, 0, false, true)
	dbCred := db{fmt.Sprintf("%s_%s", wp.AppName, randInt), fmt.Sprintf("%s_%s", wp.AppName, randInt), pass}

	err = createDatabase(dbCred, f)
	if err != nil {
		result := &errcode{
			Code:    103,
			Message: "Cannot create database",
		}
		return c.JSON(http.StatusBadRequest, result)
	}

	//Create folder in user home directory for wordpress
	path = fmt.Sprintf("/home/%s/%s", wp.UserName, wp.AppName)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir %s", path)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir %s/{public,private}", path)).Output()

	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("chown -R %s:%s %s", wp.UserName, wp.UserName, path)).Output()
	if err != nil {
		result := &errcode{
			Code:    104,
			Message: "cannot create folder for wordpress",
		}
		return c.JSON(http.StatusBadRequest, result)
	}
	path = path + "/public"
	// Download wordpress
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli core download --path=%s", wp.UserName, path)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		f.Write(write)
		f.Write(out)
		f.Close()
		result := &errcode{
			Code:    105,
			Message: "Cannot download wordpress",
		}
		return c.JSON(http.StatusBadRequest, result)
	}

	// Create config file with database crediantls for DB struct
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli config create --path=%s --dbname=%s --dbuser=%s --dbpass=%s", wp.UserName, path, dbCred.Name, dbCred.User, dbCred.Password)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		f.Write(write)
		f.Write(out)
		f.Close()
		result := &errcode{
			Code:    106,
			Message: "Connot configure wp-config file",
		}
		return c.JSON(http.StatusBadRequest, result)
	}
	// 	f, err := os.OpenFile(fmt.Sprintf("%s/wp-config.php", path), os.O_APPEND|os.O_WRONLY, 0644)
	// 	if err != nil {
	// 		return echo.NewHTTPError(http.StatusBadRequest, "cannot add http block to wpconfig file")
	// 	}

	// 	f.WriteString(`
	// /*######################################################################
	// ######################################################################
	// ###        DO NOT REMOVE THIS BLOCK. ADDED BY HOSTING            #####*/
	// if ($_SERVER['HTTP_X_FORWARDED_PROTO'] === 'https') {
	// 	$_SERVER['HTTPS'] = 'on';
	// }
	// /*######################################################################*/`)

	// 	f.Close()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("touch %s/.htaccess", path)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("echo \" %s/.htaccess IN_MODIFY /usr/sbin/service lsws restart\" >> /etc/incron.d/sites.txt", path)).Output()
	exec.Command("/bin/bash", "-c", "incrontab /etc/incron.d/sites.txt").Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("chown %s:%s %s/.htaccess", wp.UserName, wp.UserName, path)).Output()

	//Add phpini file
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir -p /usr/local/lsws/php-ini/%s", wp.AppName))
	phpfile, _ := os.OpenFile(fmt.Sprintf("/usr/local/lsws/php-ini/%s/php.ini", wp.AppName), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	phpfile.Write([]byte(`
	[PHP]
	max_execution_time=200
	max_file_uploads=20
	max_input_time=60
	max_input_vars=2000
	memory_limit=256M
	post_max_size=512M
	session.cookie_lifetime=0
	session.gc_maxlifetime=1440
	upload_max_filesize=512M
	`))
	phpfile.Close()
	// Install wordpress with data provided by request
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli core install --path=%s --url=%s --title=%s --admin_user=%s --admin_password=%s --admin_email=%s", wp.UserName, path, wp.Url, wp.Title, wp.AdminUser, wp.AdminPassword, wp.AdminEmail)).CombinedOutput()

	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		f.Write(write)
		f.Write(out)
		f.Close()
		result := &errcode{
			Code:    107,
			Message: "Cannot install wordpress",
		}
		return c.JSON(http.StatusBadRequest, result)
	}
	err = editLsws(*wp)
	if err != nil {
		result := &errcode{
			Code:    108,
			Message: "Edit lsws error",
		}
		return c.JSON(http.StatusBadRequest, result)
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir -p /var/logs/Hosting/%s", wp.AppName))

	err = addSiteToJSON(*wp, "live")
	if err != nil {
		result := &errcode{
			Code:    109,
			Message: "Error occured while adding site to json",
		}
		return c.JSON(http.StatusBadRequest, result)
	}

	err = configNuster()
	if err != nil {
		result := &errcode{
			Code:    110,
			Message: "Error occured while configuring nuster",
		}
		return c.JSON(http.StatusBadRequest, result)
	}
	exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir -p /var/log/hosting/%s", wp.AppName)).Output()
	return c.JSON(http.StatusOK, dbCred)

}

func wpDelete(c echo.Context) error {
	wp := new(wpdelete)
	c.Bind(&wp)
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", wp.Main.User, wp.Main.Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 || len(dbnameArray) == 0 {
		return c.JSON(404, "invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_USER | cut -d \\' -f 4", wp.Main.User, wp.Main.Name)).Output()
	dbuser := strings.TrimSuffix(string(db), "\n")
	dbuserArray := strings.Split(dbuser, "\n")
	if len(dbuserArray) > 1 || len(dbuserArray) == 0 {
		return c.JSON(404, "invalid wp-config file")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s", wp.Main.User, wp.Main.Name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /usr/local/lsws/conf/vhosts/%s.*", wp.Main.Name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"DROP DATABASE %s;\"", dbname)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"DROP USER '%s'@'localhost';\"", dbuser)).Output()
	// exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/ondemand --password=kopia ; kopia snapshot delete --all-snapshots-for-source /home/%s/%s --delete", user, name)).Output()
	deleteSiteFromJSON(wp.Main.Name)
	log.Print("Checking if staging is true")
	log.Print(fmt.Sprintf("Staging is %t", wp.IsStaging))
	if wp.IsStaging {
		deleteStagingSiteInternal(wp.Staging.Name, wp.Staging.User)
	} else {
		go exec.Command("/bin/bash", "-c", "killall lsphp").Output()
		go exec.Command("/bin/bash", "-c", "service lsws restart").Output()

		configNuster()

		go exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	}

	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/%s\\/%s/d' /etc/incron.d/sites.txt", wp.Main.User, wp.Main.Name)).Output()
	return c.String(http.StatusOK, "Delete success")
}

func createDatabase(d db, f *os.File) error {
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"CREATE DATABASE %s;\"", d.Name)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(d, "", "  ")
		f.Write(write)
		f.Write(out)
		f.Close()
		return echo.NewHTTPError(http.StatusBadRequest, string(out))
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"CREATE USER '%s'@'localhost' IDENTIFIED BY '%s';\"", d.User, d.Password)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(d, "", "  ")
		f.Write(write)
		f.Write(out)
		f.Close()
		return echo.NewHTTPError(http.StatusBadRequest, string(out))
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost';\"", d.Name, d.User)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(d, "", "  ")
		f.Write(write)
		f.Write(out)
		f.Close()
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	exec.Command("/bin/bash", "-c", "mysql -e 'FLUSH PRIVILEGES;'").Output()

	return nil
}

func getPluginAndThemesStatus(c echo.Context) error {
	user := c.Param("user")
	name := c.Param("name")
	plugin, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin list --format=json --path='/home/%[1]s/%[2]s/public'", user, name)).CombinedOutput()
	if err != nil {
		return c.JSON(404, "Cannot get plugins list")
	}
	theme, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli theme list --format=json --path='/home/%[1]s/%[2]s/public'", user, name)).Output()
	if err != nil {
		return c.JSON(404, "Cannot get themes list")
	}
	var plugins []interface{}
	err = json.Unmarshal(plugin, &plugins)
	if err != nil {
		fmt.Println(err)
		return c.JSON(400, "err")
	}
	var themes []interface{}
	err = json.Unmarshal(theme, &themes)
	if err != nil {
		fmt.Println(err)
		return c.JSON(400, "err")
	}
	return c.JSON(200, map[string]interface{}{"plugins": plugins, "themes": themes})
}

func updatePluginsThemes(c echo.Context) error {
	user := c.Param("user")
	name := c.Param("name")
	var body = new(PluginsThemesOperation)
	c.Bind(&body)
	result := make(map[string]int)
	for _, item := range body.Plugins {
		switch item.Operation {
		case "update":
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin update %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				result[item.Name] = 0
			} else {
				result[item.Name] = 1

			}

		case "activate":
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin activate %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				result[item.Name] = 0
			} else {
				result[item.Name] = 1

			}

		case "deactivate":
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin deactivate %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				result[item.Name] = 0
			} else {
				result[item.Name] = 1

			}

		}
	}
	for _, item := range body.Themes {
		switch item.Operation {
		case "update":

			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli theme update %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				result[item.Name] = 0
			} else {
				result[item.Name] = 1

			}

		case "activate":
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli theme activate %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				result[item.Name] = 0
			} else {
				result[item.Name] = 1
			}
		}
	}

	return c.JSON(200, result)
}
