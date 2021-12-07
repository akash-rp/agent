package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func addSiteToJSON(wp wpadd, siteType string) error {
	// read file
	// data, err := ioutil.ReadFile("/usr/Hosting/config.json")
	// if err != nil {
	// 	return echo.NewHTTPError(404, "Config file not found")
	// }

	// // json data
	// var obj Config

	// // unmarshall it
	// err = json.Unmarshal(data, &obj)
	// if err != nil {
	// 	return echo.NewHTTPError(400, "JSON data error")
	// }
	newSite := Site{Name: wp.AppName, Cache: "off", User: wp.UserName, Type: siteType}
	newSite.AliasDomain = []Domain{}

	newSite.PrimaryDomain = Domain{Url: wp.Url, SSL: false, WildCard: false}
	obj.Sites = append(obj.Sites, newSite)
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	return nil
}

func deleteSiteFromJSON(appName string) error {
	// data, err := ioutil.ReadFile("/usr/Hosting/config.json")
	// if err != nil {
	// 	return echo.NewHTTPError(404, "Config file not found")
	// }

	// // json data
	// var obj Config

	// // unmarshall it
	// err = json.Unmarshal(data, &obj)
	// if err != nil {
	// 	return echo.NewHTTPError(400, "JSON data error")
	// }

	for i, site := range obj.Sites {
		if site.Name == appName {
			obj.Sites = RemoveIndex(obj.Sites, i)
		}
	}

	if len(obj.Sites) == 0 {
		obj.SSL = false
	}

	for _, site := range obj.Sites {
		if site.PrimaryDomain.SSL {
			obj.SSL = true
			break
		}
		obj.SSL = false
	}

	back, _ := json.MarshalIndent(obj, "", "  ")
	err := ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	if err != nil {
		return echo.NewHTTPError(400, "Cannot write to config file")
	}
	return nil
}

func getSites(c echo.Context) error {

	return c.JSON(http.StatusOK, &obj.Sites)
}

func RemoveIndex(s []Site, index int) []Site {
	return append(s[:index], s[index+1:]...)
}

func addCert(wp wpcert) error {

	data, _ := ioutil.ReadFile("/usr/Hosting/config.json")

	// json data
	var obj Config

	// unmarshall it
	err := json.Unmarshal(data, &obj)
	if err != nil {
		return echo.NewHTTPError(400, "JSON data error")
	}

	for i, site := range obj.Sites {
		if wp.AppName == site.Name {
			if wp.Url == site.PrimaryDomain.Url {
				_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("service hosting stop; certbot certonly --standalone --dry-run -d %s", wp.Url)).Output()
				if err != nil {
					return echo.NewHTTPError(404, "Error with cert config")
				}
				_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("certbot certonly --standalone -d %s -m %s --agree-tos --cert-name %s", wp.Url, wp.Email, wp.Url)).Output()
				if err != nil {
					return echo.NewHTTPError(404, "Error with cert config after dry run")
				}
				obj.SSL = true
				exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /etc/letsencrypt/live/%s/fullchain.pem /etc/letsencrypt/live/%s/privkey.pem > /opt/Hosting/certs/%s.pem", wp.Url, wp.Url, wp.Url))
				obj.Sites[i].PrimaryDomain.SSL = true
				back, _ := json.MarshalIndent(obj, "", "  ")
				err = ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
				if err != nil {
					return echo.NewHTTPError(400, "Cannot write to config file")
				}
				configNuster()
				exec.Command("/bin/bash", "-c", "service hosting start")
				return nil
			}
		}
	}

	return echo.NewHTTPError(404, "Domain not found with this app")
}

// define data structure

func SaveJSONFile() error {
	back, _ := json.MarshalIndent(obj, "", "  ")
	err := ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	if err != nil {
		return errors.New("cannot save JSON File")
	}
	return nil
}
