package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/sethvargo/go-password/password"
)

func main() {
	err := configNuster()
	e := echo.New()
	if err != nil {
		e.Logger.Fatal(err)
	}
	e.GET("/serverstats", serverStats)
	e.POST("/wp/add", wpAdd)
	e.POST("/wp/delete", wpDelete)
	e.GET("/hositng", hosting)
	e.Logger.Fatal(e.Start(":8081"))
}

func serverStats(c echo.Context) error {
	totalmem, err := exec.Command("/bin/bash", "-c", "free -m | awk 'NR==2{printf $2}'").Output()
	usedmem, err := exec.Command("/bin/bash", "-c", "free -m | awk 'NR==2{printf $3}'").Output()
	cores, err := exec.Command("/bin/bash", "-c", "nproc").Output()
	cpuname, err := exec.Command("/bin/bash", "-c", "lscpu | grep 'Model name' | cut -f 2 -d : | awk '{$1=$1}1'").Output()
	os, err := exec.Command("/bin/bash", "-c", "hostnamectl | grep 'Operating System' | cut -f 2 -d : | awk '{$1=$1}1'").Output()
	if err != nil {
		log.Fatal(err)
	}

	m := &systemstats{
		TotalMemory: string(totalmem),
		UsedMemory:  string(usedmem),
		Cores:       strings.TrimSuffix(string(cores), "\n"),
		Cpu:         strings.TrimSuffix(string(cpuname), "\n"),
		Os:          strings.TrimSuffix(string(os), "\n"),
	}
	return c.JSON(http.StatusOK, m)
}

// func wpcli(c echo.Context) error {
// 	wpdata := new(wp)
// 	err := c.Bind(&wpdata)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	return c.JSON(http.StatusOK, wpdata)
// }

func wpAdd(c echo.Context) error {
	// Bind received post request body to a struct
	wp := new(wpadd)
	c.Bind(&wp)

	// Check if all fields are defind
	if wp.AppName == "" || wp.Url == "" || wp.UserName == "" || wp.Title == "" || wp.AdminEmail == "" || wp.AdminPassword == "" || wp.AdminUser == "" {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	// check if user exists or not. If not then create a user with home directory
	_, err := user.Lookup(wp.UserName)
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
	lsByte, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("ls %s", path)).Output()
	lsStirng := string(lsByte)
	lsSlice := strings.Split(lsStirng, "\n")

	for _, ls := range lsSlice {
		if ls == wp.AppName {
			return echo.NewHTTPError(http.StatusBadRequest, "App Name exists")
		}
	}

	path = fmt.Sprintf("/home/%s/%s", wp.UserName, wp.AppName)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mkdir %s", path)).Output()
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("chown %s:%s %s", wp.UserName, wp.UserName, path)).Output()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// Create random number to concate with appName for prevention of Duplicity. Create rand password for DB password and assign them to DB struct
	randInt, _ := password.Generate(5, 5, 0, false, true)
	pass, _ := password.Generate(32, 20, 0, false, true)
	dbCred := db{fmt.Sprintf("%s_%s", wp.AppName, randInt), fmt.Sprintf("%s_%s", wp.AppName, randInt), pass}

	err = createDatabase(dbCred)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot create database")
	}

	// Download wordpress
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli core download --path=%s", wp.UserName, path)).Output()
	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		return echo.NewHTTPError(http.StatusBadRequest, "Wordpress Download Error")
	}

	// Create config file with database crediantls for DB struct
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli config create --path=%s --dbname=%s --dbuser=%s --dbpass=%s", wp.UserName, path, dbCred.Name, dbCred.User, dbCred.Password)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		return echo.NewHTTPError(http.StatusBadRequest, string(out))
	}
	// Install wordpress with data provided by request
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli core install --path=%s --url=%s --title=%s --admin_user=%s --admin_password=%s --admin_email=%s", wp.UserName, path, wp.Url, wp.Title, wp.AdminUser, wp.AdminPassword, wp.AdminEmail)).CombinedOutput()

	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		return echo.NewHTTPError(http.StatusBadRequest, string(out))
	}
	err = editLsws(*wp)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot create lsws config file")
	}

	err = addSiteToJSON(*wp)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot add site to config file")
	}

	err = configNuster()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot add site to hosting.cfg file")
	}
	exec.Command("/bin/bash", "-c", "service hosting stop").Output()
	exec.Command("/bin/bash", "-c", "service hosting start").Output()
	return c.JSON(http.StatusOK, dbCred)

}

func wpDelete(c echo.Context) error {
	wp := new(wpdelete)
	c.Bind(&wp)
	path := fmt.Sprintf("/home/%s/%s", wp.UserName, wp.AppName)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf %s", path)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"DROP DATABASE %s;\"", wp.DbName)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"DROP USER '%s'@'localhost';\"", wp.DbUser)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /usr/local/lsws/conf/vhosts/%s.conf", wp.AppName)).Output()
	exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /usr/local/lsws/conf/vhosts/%s.d", wp.AppName)).Output()
	exec.Command("/bin/bash", "-c", "killall lsphp").Output()
	exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	path = fmt.Sprintf("/home/%s", wp.UserName)
	lsByte, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("ls %s", path)).Output()
	lsStirng := string(lsByte)
	lsSlice := strings.Split(lsStirng, "\n")
	lsSlice = lsSlice[:len(lsSlice)-1]
	shouldDelete := true
	for _, ls := range lsSlice {
		if ls != "logs" {
			shouldDelete = false
			continue
		}
	}
	if shouldDelete {
		exec.Command("/bin/bash", "-c", fmt.Sprintf("userdel -f -r %s", wp.UserName)).Output()

	}
	err := deleteSiteFromJSON(*wp)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot delete from Json file")
	}

	err = configNuster()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot config nuster file")
	}
	exec.Command("/bin/bash", "-c", "service hosting stop").Output()
	exec.Command("/bin/bash", "-c", "service hosting start").Output()

	return c.String(http.StatusOK, "Delete success")
}

func createDatabase(d db) error {
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"CREATE DATABASE %s;\"", d.Name)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(d, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		return echo.NewHTTPError(http.StatusBadRequest, string(out))
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"CREATE USER '%s'@'localhost' IDENTIFIED BY '%s';\"", d.User, d.Password)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(d, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		return echo.NewHTTPError(http.StatusBadRequest, string(out))
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("mysql -e \"GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost';\"", d.Name, d.User)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(d, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	exec.Command("/bin/bash", "-c", "mysql -e 'FLUSH PRIVILEGES;'").Output()
	if err != nil {
		write, _ := json.MarshalIndent(d, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		return echo.NewHTTPError(http.StatusBadRequest, "FLush")
	}
	return nil
}

func hosting(c echo.Context) error {
	err := configNuster()
	if err != nil {
		return err
	}
	return c.String(http.StatusOK, "Success")
}

type systemstats struct {
	Cores       string `json:"cores"`
	Cpu         string `json:"cpu"`
	TotalMemory string `json:"totalMemeory"`
	UsedMemory  string `json:"usedMemeory"`
	Os          string `json:"os"`
}

type wpadd struct {
	AppName       string `json:"appName"`
	UserName      string `json:"userName"`
	Url           string `json:"url"`
	Title         string `json:"title"`
	AdminUser     string `json:"adminUser"`
	AdminPassword string `json:"adminPassword"`
	AdminEmail    string `json:"adminEmail"`
}

type db struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type wpdelete struct {
	AppName  string `json:"appName"`
	UserName string `json:"userName"`
	DbName   string `json:"dbName"`
	DbUser   string `json:"DbUser"`
}
