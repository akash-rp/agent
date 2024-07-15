package main

import (
	"fmt"
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

	dbname, _, _, err := getDbcredentials(ChangeDomain.User, ChangeDomain.Name)
	if err != nil {
		return AbortWithErrorMessage(c, "invalid wp-config file")
	}

	rootPass, err := getMariadbRootPass()
	if err != nil {
		return AbortWithErrorMessage(c, "root password not found")
	}

	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s -u root -p %s -s http://%s -r http://%s -x guid -x user_email", dbname, rootPass, ChangeDomain.CurrentPrimary, ChangeDomain.NewPrimary)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		log.Print(err)
		return AbortWithErrorMessage(c, "search and replace operation failed")
	}

	return c.String(http.StatusOK, "success")
}

func addDomain(c echo.Context) error {
	site := new(DomainConf)
	c.Bind(&site)
	if site.Domain.Url == "" && site.SiteName == "" {
		return AbortWithErrorMessage(c, "All fields are not defined")
	}
	err := addDomainConf(site.Domain, site.SiteName)
	if err != nil {
		return AbortWithErrorMessage(c, err.Error())
	}
	addDomainToJson(*site)
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	return c.JSON(200, "Success")
}

func deleteDomain(c echo.Context) error {
	site := new(DeleteDomain)
	c.Bind(&site)
	if site.Domain == "" || site.Site == "" {
		return AbortWithErrorMessage(c, "All fields are not defined")
	}
	log.Print(site.Domain)
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf %s/%s.d/domain/%s.conf*", RootPath, site.Site, site.Domain)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		return AbortWithErrorMessage(c, "Cannot delete domain")
	}
	linuxCommand(fmt.Sprintf("rm %[1]s/%[2]s.d/domain/%[3]s.ssl* ;rm %[1]s/%[2]s.d/domain/%[3]s.rewrite*", RootPath, site.Site, site.Domain))
	deleteDomainFromJson(*site)
	defer exec.Command("/bin/bash", "-c", "service lsws restart").Output()
	return c.JSON(200, "Success")
}

func updateWildcard(c echo.Context) error {
	domain := new(UpdateWildcardResp)
	c.Bind(&domain)

	if domain.Domain.Url == "" || domain.Site == "" {
		return AbortWithErrorMessage(c, "All fields are not defined")
	}

	var url = domain.Domain.Url

	if domain.Domain.Wildcard {
		url = domain.Domain.Url + ", " + "*." + domain.Domain.Url
	} else {
		if domain.Domain.Subdomain {
			url = domain.Domain.Url
		} else {
			url = domain.Domain.Url + ", " + "www." + domain.Domain.Url
		}
	}

	_, err := os.Stat(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/domain/%s.conf", domain.Site, domain.Domain.Url))
	if err != nil {
		return c.NoContent(400)
	}
	exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/vhDomain/c\\vhDomain %s'  /usr/local/lsws/conf/vhosts/%s.d/domain/%s.conf", url, domain.Site, domain.Domain.Url)).Output()

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

func deleteDomainFromJson(conf DeleteDomain) {
	for i, site := range obj.Sites {
		if site.Name == conf.Site {
			domains := removeElementFromSlice(site.Domains, conf.Domain)
			obj.Sites[i].Domains = domains
			break
		}
	}
	SaveJSONFile()
}
