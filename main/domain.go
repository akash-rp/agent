package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

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
	// obj.Sites = Doamin.Sites
	for i, site := range obj.Sites {
		if site.Name == Domain.Name {
			obj.Sites[i].AliasDomain = Domain.Site.AliasDomain
			obj.Sites[i].PrimaryDomain = Domain.Site.PrimaryDomain
		}
	}
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

	go exec.Command("/bin/bash", "-c", "service lshttpd restart").Output()

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
	go exec.Command("/bin/bash", "-c", "service hosting restart").Output()
	return c.String(http.StatusOK, "success")
}

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

	for i, site := range obj.Sites {
		if site.Name == ChangeDomain.Name {
			prim := site.PrimaryDomain
			var alias Domain
			for ia, ali := range site.AliasDomain {
				if ali.Url == ChangeDomain.MainUrl {
					alias = ali
					site.AliasDomain[ia] = prim
				}
			}
			site.PrimaryDomain = alias
		}
		obj.Sites[i] = site
	}
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	db, _ := exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /home/%s/%s/public/wp-config.php | grep DB_NAME | cut -d \\' -f 4", ChangeDomain.User, ChangeDomain.Name)).Output()
	dbname := strings.TrimSuffix(string(db), "\n")
	dbnameArray := strings.Split(dbname, "\n")
	if len(dbnameArray) > 1 {
		return errors.New("invalid wp-config file")
	}
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("php /usr/Hosting/script/srdb.cli.php -h localhost -n %s -u root -p '' -s http://%s -r http://%s -x guid -x user_email", dbnameArray[0], ChangeDomain.AliasUrl, ChangeDomain.MainUrl)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		log.Print(err)
		return errors.New("search and replace operation failed")
	}
	return c.String(http.StatusOK, "success")
}
