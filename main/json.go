package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

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

// define data structure

func SaveJSONFile() error {
	back, _ := json.MarshalIndent(obj, "", "  ")
	err := ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
	if err != nil {
		return errors.New("cannot save JSON File")
	}
	return nil
}
