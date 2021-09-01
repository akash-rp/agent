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
	"gopkg.in/ini.v1"
)

var obj Config

func main() {
	e := echo.New()
	data, _ := ioutil.ReadFile("/usr/Hosting/config.json")
	json.Unmarshal(data, &obj)
	configNuster()
	log.Print("8:13AM")
	e.GET("/serverstats", serverStats)
	e.POST("/wp/add", wpAdd)
	e.POST("/wp/delete", wpDelete)
	e.GET("/hositng", hosting)
	e.POST("/cert", cert)
	e.GET("/sites", getSites)
	e.POST("/domainedit", editDomain)
	e.POST("/changeprimary", changePrimary)
	e.POST("/changePHP", changePHP)
	e.GET("/getPHPini/:name", getPHPini)
	e.POST("/updatePHPini/:name", updatePHPini)
	e.Logger.Fatal(e.Start(":8081"))
}

func serverStats(c echo.Context) error {
	totalmem, _ := exec.Command("/bin/bash", "-c", "free -m | awk 'NR==2{printf $2}'").Output()
	usedmem, _ := exec.Command("/bin/bash", "-c", "free -m | awk 'NR==2{printf $3}'").Output()
	cores, _ := exec.Command("/bin/bash", "-c", "nproc").Output()
	cpuname, _ := exec.Command("/bin/bash", "-c", "lscpu | grep 'Model name' | cut -f 2 -d : | awk '{$1=$1}1'").Output()
	totaldisk, _ := exec.Command("/bin/bash", "-c", " df -h --total -x tmpfs | awk '/total/{printf $2}'").Output()
	useddisk, _ := exec.Command("/bin/bash", "-c", " df -h --total -x tmpfs| awk '/total/{printf $3}'").Output()
	bandwidth, _ := exec.Command("/bin/bash", "-c", "vnstat | awk 'NR==4{print $5$6}'").Output()
	os, err := exec.Command("/bin/bash", "-c", "hostnamectl | grep 'Operating System' | cut -f 2 -d : | awk '{$1=$1}1'").Output()
	if err != nil {
		log.Fatal(err)
	}
	stringBandwith := string(bandwidth)
	stringBandwith = strings.ReplaceAll(stringBandwith, "i", "")
	m := &systemstats{
		TotalMemory: string(totalmem),
		UsedMemory:  string(usedmem),
		TotalDisk:   string(totaldisk),
		UsedDisk:    string(useddisk),
		Bandwidth:   strings.TrimSuffix(stringBandwith, "\n"),
		Cores:       strings.TrimSuffix(string(cores), "\n"),
		Cpu:         strings.TrimSuffix(string(cpuname), "\n"),
		Os:          strings.TrimSuffix(string(os), "\n"),
	}
	return c.JSON(http.StatusOK, m)
}

func wpAdd(c echo.Context) error {
	// Bind received post request body to a struct
	wp := new(wpadd)
	c.Bind(&wp)

	// Check if all fields are defind
	if wp.AppName == "" || wp.Url == "" || wp.UserName == "" || wp.Title == "" || wp.AdminEmail == "" || wp.AdminPassword == "" || wp.AdminUser == "" {
		result := &errcode{
			Code:    101,
			Message: "Required fields are not defined",
		}
		return c.JSON(http.StatusBadRequest, result)
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

	err = createDatabase(dbCred)
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
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("chown %s:%s %s", wp.UserName, wp.UserName, path)).Output()
	if err != nil {
		result := &errcode{
			Code:    104,
			Message: "cannot create folder for wordpress",
		}
		return c.JSON(http.StatusBadRequest, result)
	}

	// Download wordpress
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli core download --path=%s", wp.UserName, path)).Output()
	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
		result := &errcode{
			Code:    105,
			Message: "Cannot download wordpress",
		}
		return c.JSON(http.StatusBadRequest, result)
	}

	// Create config file with database crediantls for DB struct
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli config create --path=%s --dbname=%s --dbuser=%s --dbpass=%s", wp.UserName, path, dbCred.Name, dbCred.User, dbCred.Password)).CombinedOutput()
	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
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
	// Install wordpress with data provided by request
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli core install --path=%s --url=%s --title=%s --admin_user=%s --admin_password=%s --admin_email=%s", wp.UserName, path, wp.Url, wp.Title, wp.AdminUser, wp.AdminPassword, wp.AdminEmail)).CombinedOutput()

	if err != nil {
		write, _ := json.MarshalIndent(dbCred, "", "  ")
		ioutil.WriteFile("/usr/Hosting/error.log", write, 0777)
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

	err = addSiteToJSON(*wp)
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

	err := deleteSiteFromJSON(*wp)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot delete from Json file")
	}

	err = configNuster()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot config nuster file")
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/%s\\/%s/d' /etc/incron.d/sites.txt", wp.UserName, wp.AppName)).Output()
	exec.Command("/bin/bash", "-c", "service hosting restart").Output()

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
	exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	return c.String(http.StatusOK, "Success")
}

func cert(c echo.Context) error {
	wp := new(wpcert)
	c.Bind(&wp)

	err := addCert(*wp)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	return c.String(http.StatusOK, "Success")
}

func editDomain(c echo.Context) error {
	Domain := new(DomainEdit)
	c.Bind(&Domain)
	// data, err := ioutil.ReadFile("/usr/Hosting/config.json")
	// if err != nil {
	// 	return echo.NewHTTPError(404, "Config file not found")
	// }
	// var obj Config

	// // unmarshall it
	// err = json.Unmarshal(data, &obj)
	// if err != nil {
	// 	return echo.NewHTTPError(400, "JSON data error")
	// }

	obj.Sites = Domain.Sites
	siteArray := []string{}
	path := ""
	for _, site := range obj.Sites {
		if site.Name == Domain.Name {
			path = fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/main.conf", site.Name)
			siteArray = append(siteArray, site.PrimaryDomain.Url)
			if site.PrimaryDomain.WildCard {
				siteArray = append(siteArray, "*."+site.PrimaryDomain.Url)
			}
			for _, ali := range site.AliasDomain {
				siteArray = append(siteArray, ali.Url)
				if ali.WildCard {
					siteArray = append(siteArray, "*."+ali.Url)
				}
			}
			break
		}
	}
	siteString := strings.Join(siteArray, ",")
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/VhDomain.*/VhDomain %s/' %s", siteString, path)).Output()

	exec.Command("/bin/bash", "-c", "service lshttpd restart").Output()

	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	err := configNuster()
	if err != nil {
		result := &errcode{
			Code:    110,
			Message: "Error occured while configuring nuster",
		}
		return c.JSON(http.StatusBadRequest, result)
	}
	exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	return c.String(http.StatusOK, "success")
}

func changePrimary(c echo.Context) error {
	Domain := new(PrimaryChange)
	c.Bind(&Domain)
	// data, err := ioutil.ReadFile("/usr/Hosting/config.json")
	// if err != nil {
	// 	return echo.NewHTTPError(404, "Config file not found")
	// }
	// var obj Config

	// // unmarshall it
	// err = json.Unmarshal(data, &obj)
	// if err != nil {
	// 	return echo.NewHTTPError(400, "JSON data error")
	// }

	obj.Sites = Domain.Sites
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	path := fmt.Sprintf("/home/%s/%s", Domain.User, Domain.Name)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %s -i -- /usr/Hosting/wp-cli search-replace '%s' '%s' --path='%s' --skip-columns=guid ", Domain.User, Domain.AliasUrl, Domain.MainUrl, path)).Output()
	return c.String(http.StatusOK, "success")

}

func changePHP(c echo.Context) error {
	PHPDetails := new(PHPChange)
	c.Bind(&PHPDetails)
	// data, err := ioutil.ReadFile("/usr/Hosting/config.json")
	// if err != nil {
	// 	return echo.NewHTTPError(404, "Config file not found")
	// }
	// var obj Config

	// // unmarshall it
	// err = json.Unmarshal(data, &obj)
	// if err != nil {
	// 	return echo.NewHTTPError(400, "JSON data error")
	// }
	obj.Sites = PHPDetails.Sites
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s|path /usr/local/lsws/%s/bin/lsphp|path /usr/local/lsws/%s/bin/lsphp|' /usr/local/lsws/conf/vhosts/%s.d/handlers/extphp.conf", PHPDetails.OldPHP, PHPDetails.NewPHP, PHPDetails.Name)).Output()
	exec.Command("/bin/bash", "-c", "service lshttpd restart").Start()
	exec.Command("/bin/bash", "-c", "killall lsphp").Start()
	return c.String(http.StatusOK, "success")

}

func getPHPini(c echo.Context) error {
	name := c.Param("name")
	path := fmt.Sprintf("/usr/local/lsws/php-ini/%s-php.ini", name)
	cfg, _ := ini.Load(path)
	var php PHP
	cfg.Section("PHP").MapTo(&php)
	return c.JSON(http.StatusOK, php)
}

func updatePHPini(c echo.Context) error {
	php := new(PHPini)
	c.Bind(&php)
	name := c.Param("name")
	cfg := ini.Empty()
	ini.ReflectFrom(cfg, php)
	cfg.SaveTo(fmt.Sprintf("/usr/local/lsws/php-ini/%s-php.ini", name))
	exec.Command("/bin/bash", "-c", "service lshttpd restart").Start()
	exec.Command("/bin/bash", "-c", "killall lsphp").Start()
	return c.JSON(http.StatusOK, "success")
}

type systemstats struct {
	Cores       string `json:"cores"`
	Cpu         string `json:"cpu"`
	TotalMemory string `json:"totalMemory"`
	UsedMemory  string `json:"usedMemory"`
	TotalDisk   string `json:"totalDisk"`
	UsedDisk    string `json:"usedDisk"`
	Bandwidth   string `json:"bandwidth"`
	Os          string `json:"os"`
}

type wpadd struct {
	AppName       string   `json:"appName"`
	UserName      string   `json:"userName"`
	Url           string   `json:"url"`
	Title         string   `json:"title"`
	AdminUser     string   `json:"adminUser"`
	AdminPassword string   `json:"adminPassword"`
	AdminEmail    string   `json:"adminEmail"`
	SubDomain     bool     `json:"subdomain"`
	Routing       string   `json:"routing"`
	Sites         []Site   `json:"sites"`
	Exclude       []string `json:"exclude"`
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

type wpcert struct {
	AppName string `json:"appName"`
	Url     string `json:"url"`
	Email   string `json:"email"`
}

type errcode struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type DomainEdit struct {
	Name  string `json:"name"`
	Sites []Site `json:"sites"`
}

type PrimaryChange struct {
	Name     string `json:"name"`
	Sites    []Site `json:"sites"`
	MainUrl  string `json:"mainUrl"`
	AliasUrl string `json:"aliasUrl"`
	User     string `json:"user"`
}

type PHPChange struct {
	Name   string `json:"name"`
	Sites  []Site `json:"sites"`
	OldPHP string `json:"oldphp"`
	NewPHP string `json:"newphp"`
}

type PHP struct {
	MaxExecutionTime      string `ini:"max_execution_time"`
	MaxFileUploads        string `ini:"max_file_uploads"`
	MaxInputTime          string `ini:"max_input_time"`
	MaxInputVars          string `ini:"max_input_vars"`
	MemoryLimit           string `ini:"memory_limit"`
	PostMaxSize           string `ini:"post_max_size"`
	SessionCookieLifetime string `ini:"session.cookie_lifetime"`
	SessionGcMaxlifetime  string `ini:"session.gc_maxlifetime"`
	ShortOpenTag          string `ini:"short_open_tag"`
	UploadMaxFilesize     string `ini:"upload_max_filesize"`
}

type PHPini struct {
	PHP
}
