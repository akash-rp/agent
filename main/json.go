package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/labstack/echo/v4"
)

func addSiteToJSON(AppName string, UserName string, url string, siteType string) error {
	newSite := Site{Name: AppName, Cache: "off", User: UserName, Type: siteType}

	newSite.Domains = append(newSite.Domains, url)
	obj.Sites = append(obj.Sites, newSite)
	back, _ := json.MarshalIndent(obj, "", "  ")
	ioutil.WriteFile("/usr/Hosting/config.json", back, 0700)
	return nil
}

func deleteSiteFromJSON(appName string) error {
	for i, site := range obj.Sites {
		if site.Name == appName {
			obj.Sites = RemoveIndex(obj.Sites, i)
		}
	}

	back, _ := json.MarshalIndent(obj, "", "  ")
	err := ioutil.WriteFile("/usr/Hosting/config.json", back, 0700)
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
	err := ioutil.WriteFile("/usr/Hosting/config.json", back, 0700)
	if err != nil {
		return errors.New("cannot save JSON File")
	}
	return nil
}

func removeElementFromSlice(slice []string, element string) []string {
	for i, item := range slice {
		if item == element {
			slice[i] = slice[len(slice)-1] // Copy last element to index i.
			slice[len(slice)-1] = ""       // Erase last element (write zero value).
			slice = slice[:len(slice)-1]
			break
		}
	}
	return slice

}
