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

	"github.com/labstack/echo/v4"
)

// func editDomain(c echo.Context) error {
// 	Domain := new(DomainEdit)
// 	c.Bind(&Domain)
// 	// data, err := ioutil.ReadFile("/usr/Hosting/config.json")
// 	// if err != nil {
// 	// 	return echo.NewHTTPError(404, "Config file not found")
// 	// }
// 	// var obj Config

// 	// // unmarshall it
// 	// err = json.Unmarshal(data, &obj)
// 	// if err != nil {
// 	// 	return echo.NewHTTPError(400, "JSON data error")
// 	// }
// 	// obj.Sites = Doamin.Sites
// 	for i, site := range obj.Sites {
// 		if site.Name == Domain.Name {
// 			obj.Sites[i].AliasDomain = Domain.Site.AliasDomain
// 			obj.Sites[i].PrimaryDomain = Domain.Site.PrimaryDomain
// 		}
// 	}
// 	siteArray := []string{}
// 	path := ""
// 	for _, site := range obj.Sites {
// 		if site.Name == Domain.Name {

// 			path = fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/main.conf", site.Name)
// 			siteArray = append(siteArray, site.PrimaryDomain.Url)
// 			if site.PrimaryDomain.WildCard {
// 				siteArray = append(siteArray, "*."+site.PrimaryDomain.Url)
// 			}
// 			for _, ali := range site.AliasDomain {
// 				siteArray = append(siteArray, ali.Url)
// 				if ali.WildCard {
// 					siteArray = append(siteArray, "*."+ali.Url)
// 				}
// 			}
// 			break
// 		}
// 	}
// 	siteString := strings.Join(siteArray, ",")
// 	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/^#/!s/VhDomain.*/VhDomain %s/' %s", siteString, path)).Output()

// 	go exec.Command("/bin/bash", "-c", "service lsws restart").Output()

// 	back, _ := json.MarshalIndent(obj, "", "  ")
// 	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)

// 	return c.String(http.StatusOK, "success")
// }

func changePrimary(c echo.Context) error {
	ChangeDomain := new(PrimaryChange)
	c.Bind(&ChangeDomain)
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

	// for i, site := range obj.Sites {
	// 	if site.Name == ChangeDomain.Name {
	// 		prim := site.PrimaryDomain
	// 		var alias DomainJSON
	// 		for ia, ali := range site.AliasDomain {
	// 			if ali.Url == ChangeDomain.MainUrl {
	// 				alias = ali
	// 				site.AliasDomain[ia] = prim
	// 			}
	// 		}
	// 		site.PrimaryDomain = alias
	// 	}
	// 	obj.Sites[i] = site
	// }
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	dbname, _, _, err := getDbcredentials(ChangeDomain.User, ChangeDomain.Name)
	if err != nil {
		return errors.New("invalid wp-config file")
	}
	rootPass, err := getMariadbRootPass()
	if err != nil {
		return errors.New("root password not found")
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s -u root -p %s -s http://%s -r http://%s -x guid -x user_email", dbname, rootPass, ChangeDomain.AliasUrl, ChangeDomain.MainUrl)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		log.Print(err)
		return errors.New("search and replace operation failed")
	}
	return c.String(http.StatusOK, "success")
}

func addDomain(c echo.Context) error {
	site := new(DomainConf)
	c.Bind(&site)
	if site.Domain.Url == "" && site.SiteName == "" {
		return c.JSON(400, "All fields are not defined")
	}
	err := addDomainConf(site.Domain, site.SiteName)
	if err != nil {
		return c.JSON(400, err)
	}
	addDomainToJson(*site)
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	return c.JSON(200, "Success")
}

func deleteDomain(c echo.Context) error {
	site := new(DomainConf)
	c.Bind(&site)
	if site.Domain.Url == "" || site.SiteName == "" {
		return c.JSON(400, "All fields are not defined")
	}
	log.Print(site.Domain.Url)
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf %s/%s.d/domain/%s.conf*", RootPath, site.SiteName, site.Domain.Url)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		return c.JSON(400, "Cannot delete domain")
	}
	linuxCommand(fmt.Sprintf("rm %[1]s/%[2]s.d/domain/%[3]s.ssl* ;rm %[1]s/%[2]s.d/domain/%[3]s.rewrite*", RootPath, site.SiteName, site.Domain.Url))
	deleteDomainFromJson(*site)
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	return c.JSON(200, "Success")
}

func addWildcard(c echo.Context) error {
	site := new(DomainConf)
	c.Bind(&site)
	if site.Domain.Url == "" || site.SiteName == "" {
		return c.JSON(400, "All fields are not defined")
	}
	domain := site.Domain.Url + ", " + "*." + site.Domain.Url
	_, err := os.Stat(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/domain/%s.conf", site.SiteName, site.Domain.Url))
	if err != nil {
		return c.NoContent(400)
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/vhDomain/c\\vhDomain %s'  /usr/local/lsws/conf/vhosts/%s.d/domain/%s.conf", domain, site.SiteName, site.Domain.Url)).Output()

	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	return c.NoContent(200)
}

func removeWildcard(c echo.Context) error {
	site := new(DomainConf)
	c.Bind(&site)
	if site.Domain.Url == "" || site.SiteName == "" {
		return c.JSON(400, "All fields are not defined")
	}
	var domain string
	if site.Domain.IsSubDomain {
		domain = site.Domain.Url
	} else {
		domain = site.Domain.Url + ", " + "www." + site.Domain.Url
	}
	_, err := os.Stat(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/domain/%s.conf", site.SiteName, site.Domain.Url))
	if err != nil {
		return c.NoContent(400)
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/vhDomain/c\\vhDomain %s'  /usr/local/lsws/conf/vhosts/%s.d/domain/%s.conf", domain, site.SiteName, site.Domain.Url)).Output()

	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	return c.NoContent(200)
}

func addDomainToJson(conf DomainConf) {
	for i, site := range obj.Sites {
		if site.Name == conf.SiteName {

			site.Domains = append(site.Domains, conf.Domain.Url)
			obj.Sites[i] = site
		}
	}
	SaveJSONFile()
}

func deleteDomainFromJson(conf DomainConf) {
	for i, site := range obj.Sites {
		if site.Name == conf.SiteName {
			domains := removeElementFromSlice(site.Domains, conf.Domain.Url)
			obj.Sites[i].Domains = domains
			break
		}
	}
	SaveJSONFile()
}
