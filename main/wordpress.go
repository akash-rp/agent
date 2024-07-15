package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sethvargo/go-password/password"
	"golang.org/x/crypto/bcrypt"
)

func wpAdd(c echo.Context) error {
	// Bind received post request body to a struct
	wp := new(wpadd)
	c.Bind(&wp)
	f, err := os.OpenFile("/usr/Hosting/error.log", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		AbortWithErrorMessage(c, "Failed to open log file")
	}
	defer f.Close()
	if err != nil {
		return c.JSON(404, err)
	}

	// Check if all fields are defind
	if wp.AppName == "" || wp.Domain.Url == "" || wp.UserName == "" || wp.Title == "" || wp.AdminEmail == "" || wp.AdminPassword == "" || wp.AdminUser == "" {
		log.Print(wp)
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
	dbCred, err := createDatabase(wp.AppName)
	if err != nil {
		result := &errcode{
			Code:    103,
			Message: "Cannot create database",
		}
		f.Write([]byte(err.Error()))
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
	var url string
	if wp.Domain.Routing == "www" {
		url = "www." + wp.Domain.Url
	} else {
		url = wp.Domain.Url
	}
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
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir -p /usr/local/lsws/php-ini/%s", wp.AppName)).Output()
	phpfile, _ := os.OpenFile(fmt.Sprintf("/usr/local/lsws/php-ini/%s/php.ini", wp.AppName), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	phpfile.Write([]byte(fmt.Sprintf(`
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
	short_open_tag = Off
	date.timezone = "UTC"
	open_basedir="/home/%s/%s/public:/tmp"
	`, wp.UserName, wp.AppName)))
	phpfile.Close()
	// Install wordpress with data provided by request
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli core install --path=%s --url=%s --title=%s --admin_user=%s --admin_password=%s --admin_email=%s", wp.UserName, path, url, wp.Title, wp.AdminUser, wp.AdminPassword, wp.AdminEmail)).CombinedOutput()

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
	err = addNewSite(*wp)
	if err != nil {
		result := &errcode{
			Code:    108,
			Message: err.Error(),
		}
		return c.JSON(http.StatusBadRequest, result)
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir -p /var/logs/Hosting/%s", wp.AppName))

	err = addSiteToJSON(wp.AppName, wp.UserName, wp.Domain.Url, "live")
	if err != nil {
		result := &errcode{
			Code:    109,
			Message: "Error occured while adding site to json",
		}
		return c.JSON(http.StatusBadRequest, result)
	}

	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir -p /var/log/hosting/%s", wp.AppName)).Output()
	db := make(map[string]string)
	db["name"] = dbCred.Name
	db["user"] = dbCred.User
	return c.JSON(http.StatusOK, db)
}

func wpDelete(c echo.Context) error {
	wp := new(wpdelete)
	c.Bind(&wp)
	dbname, dbuser, _, err := getDbcredentials(wp.Main.User, wp.Main.Name)
	if err != nil {
		return c.JSON(404, "invalid wp-config file")
	}
	rootPass, err := getMariadbRootPass()
	if err != nil {
		return c.JSON(404, "root password not found")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%s/%s", wp.Main.User, wp.Main.Name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /usr/local/lsws/conf/vhosts/%s.*", wp.Main.Name)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -uroot -p%s -e \"DROP DATABASE %s;\"", rootPass, dbname)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -uroot -p%s -e \"DROP USER '%s'@'localhost';\"", rootPass, dbuser)).Output()
	// exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia repository connect filesystem --path=/var/Backup/ondemand --password=kopia ; kopia snapshot delete --all-snapshots-for-source /home/%s/%s --delete", user, name)).Output()
	deleteSiteFromJSON(wp.Main.Name)
	log.Print("Checking if staging is true")
	log.Printf("Staging is %t", wp.IsStaging)
	if wp.IsStaging {
		deleteStagingSiteInternal(wp.Staging.Name, wp.Staging.User)
	} else {
		defer exec.Command("/bin/bash", "-c", "killall lsphp").Output()
		defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()

	}

	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/%s\\/%s/d' /etc/incron.d/sites.txt", wp.Main.User, wp.Main.Name)).Output()
	return c.String(http.StatusOK, "Delete success")
}

func createDatabase(AppName string) (db, error) {
	randInt, _ := password.Generate(5, 5, 0, false, true)
	pass, _ := password.Generate(32, 20, 0, false, true)
	d := db{fmt.Sprintf("%s_%s", AppName, randInt), fmt.Sprintf("%s_%s", AppName, randInt), pass}
	rootPassword, err := getMariadbRootPass()
	if err != nil {
		return db{}, echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -uroot -p%s -e \"CREATE DATABASE %s;\"", rootPassword, d.Name)).CombinedOutput()
	if err != nil {
		return db{}, errors.New(string(out))
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -uroot -p%s -e \"CREATE USER '%s'@'localhost' IDENTIFIED VIA mysql_native_password USING PASSWORD('%s');\"", rootPassword, d.User, d.Password)).CombinedOutput()
	if err != nil {
		return db{}, errors.New(string(out))
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -uroot -p%s -e \"GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost';\"", rootPassword, d.Name, d.User)).CombinedOutput()

	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -uroot -p%s -e 'FLUSH PRIVILEGES;'", rootPassword)).Output()

	return d, nil
}

func getPluginsList(c echo.Context) error {
	user := c.Param("user")
	name := c.Param("name")
	plugin, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin list --fields=title,name,update,update_version,status,version --format=json --path='/home/%[1]s/%[2]s/public'", user, name)).Output()
	fmt.Println(string(plugin))
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(string(plugin))
		return c.JSON(404, "Cannot get plugins list")
	}
	var plugins []interface{}
	err = json.Unmarshal(plugin, &plugins)
	if err != nil {
		fmt.Println(err)
		return AbortWithErrorMessage(c, "err")
	}
	return c.JSON(200, plugins)
}

func getThemesList(c echo.Context) error {
	user := c.Param("user")
	name := c.Param("name")
	theme, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli theme list --format=json --path='/home/%[1]s/%[2]s/public' --fields=title,name,update,update_version,status,version", user, name)).Output()
	fmt.Println(string(theme))
	if err != nil {
		return c.JSON(404, "Cannot get themes list")
	}

	var themes []interface{}
	err = json.Unmarshal(theme, &themes)
	if err != nil {
		fmt.Println(err)
		return AbortWithErrorMessage(c, "err")
	}

	return c.JSON(200, themes)
}

func updatePluginsThemes(c echo.Context) error {
	user := c.Param("user")
	name := c.Param("name")
	var body = new(PluginsThemesOperation)
	c.Bind(&body)
	for _, item := range body.Plugins {
		switch item.Operation {
		case "update":
			err := takeSystemBackup(name, user, fmt.Sprintf("%s Plugin Update", item.Name))
			if err != nil {
				return c.NoContent(400)
			}
			_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin update %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				return c.NoContent(400)
			}

		case "activate":
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin activate %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				return c.NoContent(400)
			}

		case "deactivate":
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin deactivate %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				return c.NoContent(400)
			}

		}
	}
	for _, item := range body.Themes {
		switch item.Operation {
		case "update":
			err := takeSystemBackup(name, user, fmt.Sprintf("%s Plugin Update", item.Name))
			if err != nil {
				return c.NoContent(400)
			}
			_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli theme update %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				return c.NoContent(400)
			}

		case "activate":
			_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli theme activate %[3]s --path='/home/%[1]s/%[2]s/public'", user, name, item.Name)).Output()
			if err != nil {
				return c.NoContent(400)
			}
		}
	}
	if len(body.Plugins) > 0 {
		return getPluginsList(c)
	} else if len(body.Themes) > 0 {
		return getThemesList(c)
	} else {
		return c.NoContent(404)
	}

}

func changeOwnership(c echo.Context) error {
	type Req struct {
		Name    string `json:"app"`
		OldUser string `json:"oldUser"`
		NewUser string `json:"newUser"`
		Backup  Backup `json:"backup"`
	}
	req := new(Req)
	c.Bind(&req)
	for cronBusy {
		time.Sleep(time.Millisecond * 100)
	}
	cronBusy = true
	if _, err := user.Lookup(req.NewUser); err != nil {
		exec.Command("/bin/bash", "-c", fmt.Sprintf("useradd --shell /bin/bash --create-home %s", req.NewUser)).Output()
	} else {
		if _, err := os.Stat(fmt.Sprintf("/home/%s", req.NewUser)); os.IsNotExist(err) {
			exec.Command("/bin/bash", "-c", fmt.Sprintf("mkhomedir_helper %s", req.NewUser)).Output()
		}
	}
	if _, err := os.Stat(fmt.Sprintf("/home/%s", req.NewUser)); os.IsNotExist(err) {
		return AbortWithErrorMessage(c, "New user path not found")
	}

	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("cp -a /home/%s/%s /home/%s/", req.OldUser, req.Name, req.NewUser)).CombinedOutput()
	log.Print(string(out))
	if err != nil {
		cronBusy = false
		return c.NoContent(400)
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("chown -R %[1]s:%[1]s /home/%[1]s/%[2]s/", req.NewUser, req.Name)).CombinedOutput()
	log.Print(string(out))
	if err != nil {
		cronBusy = false

		return c.NoContent(400)
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/automatic/automatic.config snapshot copy-history /home/%[1]s/%[2]s /home/%[3]s/%[2]s ; kopia --config-file=/var/Backup/config/ondemand/ondemand.config snapshot copy-history /home/%[1]s/%[2]s /home/%[3]s/%[2]s ; kopia --config-file=/var/Backup/config/system/system.config snapshot copy-history /home/%[1]s/%[2]s /home/%[3]s/%[2]s ;", req.OldUser, req.Name, req.NewUser)).CombinedOutput()
	log.Print(string(out))
	if err != nil {
		cronBusy = false
		return c.NoContent(400)
	}
	out, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("kopia --config-file=/var/Backup/config/automatic/automatic.config snapshot delete --all-snapshots-for-source /home/%[1]s/%[2]s --delete ; kopia --config-file=/var/Backup/config/ondemand/ondemand.config snapshot delete --all-snapshots-for-source /home/%[1]s/%[2]s --delete ; kopia --config-file=/var/Backup/config/system/system.config snapshot delete --all-snapshots-for-source /home/%[1]s/%[2]s --delete ;", req.OldUser, req.Name)).CombinedOutput()
	log.Print(string(out))

	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /home/%[1]s/%[2]s ", req.OldUser, req.Name)).CombinedOutput()
	log.Print(string(out))
	if err != nil {
		cronBusy = false
		return c.NoContent(400)
	}
	updateLocalBackup(req.Name, req.NewUser, &req.Backup)
	//change vhroot path in main.conf
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/vhRoot.*/vhRoot \\/home\\/%[1]s\\/%[2]s/' /usr/local/lsws/conf/vhosts/%[2]s.d/main.conf", req.NewUser, req.Name)).CombinedOutput()
	//change extUser and extgroup in extphp.conf
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/extUser.*/extUser %[1]s/' /usr/local/lsws/conf/vhosts/%[2]s.d/modules/extphp.conf ; sed -i '/^#/!s/extGroup.*/extGroup %[1]s/' /usr/local/lsws/conf/vhosts/%[2]s.d/modules/extphp.conf", req.NewUser, req.Name)).CombinedOutput()

	//change user in config.json
	for i, site := range obj.Sites {
		if site.Name == req.Name {
			obj.Sites[i].User = req.NewUser
			break
		}
	}
	defer SaveJSONFile()
	defer exec.Command("/bin/bash", "-c", "service lsws reload; killall lsphp").Output()
	cronBusy = false
	return c.NoContent(200)
}

func addSiteAuthentication(c echo.Context) error {
	type auth struct {
		Name string `json:"name"`
		Auth struct {
			User     string `json:"user"`
			Password string `json:"password"`
		} `json:"auth"`
	}
	req := new(auth)
	c.Bind(&req)
	pass, err := bcrypt.GenerateFromPassword([]byte(req.Auth.Password), 10)
	if err != nil {
		log.Println(err.Error())
		return AbortWithErrorMessage(c, "Failed to generate hash")
	}
	db := fmt.Sprintf("%s:%s", req.Auth.User, pass)
	err = os.WriteFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/userdb", req.Name), []byte(db), 0660)
	if err != nil {
		log.Println(err.Error())

		return AbortWithErrorMessage(c, "Failed to write userdb")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("chown nobody:nogroup /usr/local/lsws/conf/vhosts/%s.d/userdb", req.Name)).Output()
	realm := fmt.Sprintf(`
	realm auth {
		userDB  {
		  location              /usr/local/lsws/conf/vhosts/%s.d/userdb
		}
	}	  
	`, req.Name)
	err = os.WriteFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/siteauth.conf", req.Name), []byte(realm), 0660)
	if err != nil {
		log.Println(err.Error())

		return AbortWithErrorMessage(c, "Failed to write realm file")
	}
	out, _ := linuxCommand(fmt.Sprintf("sed -i 's/.*allowBrowse.*/&\\n\\trealm                   auth/' /usr/local/lsws/conf/vhosts/%s.d/modules/context.conf", req.Name))
	log.Print(string(out))
	exec.Command("/bin/bash", "-c", fmt.Sprintf("chown nobody:nogroup /usr/local/lsws/conf/vhosts/%s.d/modules/siteauth.conf", req.Name)).Output()
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	return c.NoContent(200)
}

func deleteSiteAuthentication(c echo.Context) error {
	name := c.Param("name")
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /usr/local/lsws/conf/vhosts/%[1]s.d/userdb ; rm /usr/local/lsws/conf/vhosts/%[1]s.d/modules/siteauth.conf", name)).Output()
	linuxCommand(fmt.Sprintf("sed -i '/realm/d' /usr/local/lsws/conf/vhosts/%s.d/modules/context.conf", name))
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	return c.NoContent(200)
}

func fixFilePermissionRequest(c echo.Context) error {
	type site struct {
		Name string `json:"name"`
		User string `json:"user"`
	}
	req := new(site)
	c.Bind(&req)
	if err := fixFilePermission(req.Name, req.User); err != nil {
		return c.NoContent(400)
	}
	return c.NoContent(200)
}
func fixFilePermission(Name string, User string) error {

	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("chown -R %[1]s:%[1]s /home/%[1]s/%[2]s/", User, Name)).CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return errors.New("failed to chown")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("find /home/%s/%s -type d -print0 | xargs -0 chmod 755 ", User, Name)).Output()
	if err != nil {
		log.Println(string(out))
		return errors.New("failed to chmod d")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("find /home/%s/%s -type f -print0 | xargs -0 chmod 644 ", User, Name)).Output()
	if err != nil {
		log.Println(string(out))
		return errors.New("failed to chmod f")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("chmod 604 /home/%s/%s/public/.htaccess ", User, Name)).Output()
	if err != nil {
		log.Println(string(out))
		return errors.New("failed to chmod htaccess")
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("chmod 640 /home/%s/%s/public/wp-config.php ", User, Name)).Output()
	if err != nil {
		log.Println(string(out))
		return errors.New("failed to chmod config")
	}
	return nil
}

func searchAndReplace(c echo.Context) error {
	type data struct {
		Search  string `json:"search"`
		Replace string `json:"replace"`
		Name    string `json:"name"`
		User    string `json:"user"`
	}
	Data := new(data)
	c.Bind(&Data)
	dbname, dbuser, dbpassword, err := getDbcredentials(Data.User, Data.Name)
	if err != nil {
		return AbortWithErrorMessage(c, "Failed to get database credentials")
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s -u %s -p %s -s %s -r %s", dbname, dbuser, dbpassword, Data.Search, Data.Replace)).CombinedOutput()
	log.Print(string(out))
	if err != nil {
		log.Println(string(out))
		return AbortWithErrorMessage(c, "Failed to perform serach and replace operation")
	}
	return c.JSON(200, "Success")
}

func getDbcredentials(User string, Name string) (dbname string, dbuser string, password string, err error) {
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", User, Name)).Output()
	dbOut := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbOut, "\n")
	if len(dbnameArray) > 1 || len(dbnameArray) == 0 {
		return "", "", "", errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_USER | cut -d \\' -f 4", User, Name)).Output()
	dbUser := strings.TrimSuffix(string(db), "\n")
	dbuserArray := strings.Split(dbUser, "\n")
	if len(dbuserArray) > 1 || len(dbuserArray) == 0 {
		return "", "", "", errors.New("invalid wp-config file")
	}
	db, _ = exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_PASSWORD | cut -d \\' -f 4", User, Name)).Output()
	dbpassword := strings.TrimSuffix(string(db), "\n")
	dbpasswordArray := strings.Split(dbpassword, "\n")
	if len(dbpasswordArray) > 1 || len(dbpasswordArray) == 0 {
		return "", "", "", errors.New("invalid wp-config file")
	}
	return dbnameArray[0], dbuserArray[0], dbpasswordArray[0], nil
}

func getMariadbRootPass() (password string, err error) {
	out, err := exec.Command("/bin/bash", "-c", "grep 'root' /etc/mysql/mariadb.conf.d/root.env").Output()
	if err != nil {
		return "", errors.New("root File to found")
	}
	credentials := strings.Split(strings.TrimSuffix(string(out), "\n"), ":")
	if credentials[0] == "root" {
		password := credentials[1]
		return password, nil
	}
	return "", errors.New("failed to get root password")
}
