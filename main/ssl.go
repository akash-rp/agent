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

func cert(c echo.Context) error {
	wp := new(wpcert)
	c.Bind(&wp)

	err := addCert(*wp)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	return c.String(http.StatusOK, "Success")
}

func addCert(wp wpcert) error {

	for i, site := range obj.Sites {
		if wp.AppName == site.Name {
			if wp.Url == site.PrimaryDomain.Url {
				_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("certbot certonly --standalone --dry-run -d %s -m %s --agree-tos --non-interactive --http-01-port=8888 --key-type ecdsa", wp.Url, wp.Email)).Output()
				if err != nil {
					return errors.New("error with cert config")
				}
				_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("certbot certonly --standalone -d %s -m %s --agree-tos --cert-name %s --non-interactive --http-01-port=8888 --key-type ecdsa", wp.Url, wp.Email, wp.Url)).Output()
				if err != nil {
					return errors.New("error with cert config after dry run")
				}
				obj.SSL = true
				exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /etc/letsencrypt/live/%s/fullchain.pem /etc/letsencrypt/live/%s/privkey.pem > /opt/Hosting/certs/%s.pem", wp.Url, wp.Url, wp.Url)).Output()
				obj.Sites[i].PrimaryDomain.SSL = true
				back, _ := json.MarshalIndent(obj, "", "  ")
				err = ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
				if err != nil {
					return errors.New("cannot write to config file")
				}
				configNuster()
				go exec.Command("/bin/bash", "-c", "service hosting restart").Output()
				return nil
			} else {
				for j, Domain := range site.AliasDomain {
					if wp.Url == Domain.Url {
						_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("certbot certonly --standalone --dry-run -d %s -m %s --agree-tos --non-interactive --http-01-port=8888", wp.Url, wp.Email)).Output()
						if err != nil {
							return errors.New("error with cert config")
						}
						_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("certbot certonly --standalone -d %s -m %s --agree-tos --cert-name %s --non-interactive --http-01-port=8888", wp.Url, wp.Email, wp.Url)).Output()
						if err != nil {
							return errors.New("error with cert config after dry run")
						}
						obj.SSL = true
						exec.Command("/bin/bash", "-c", fmt.Sprintf("cat /etc/letsencrypt/live/%s/fullchain.pem /etc/letsencrypt/live/%s/privkey.pem > /opt/Hosting/certs/%s.pem", wp.Url, wp.Url, wp.Url)).Output()
						obj.Sites[i].AliasDomain[j].SSL = true
						back, _ := json.MarshalIndent(obj, "", "  ")
						err = ioutil.WriteFile("/usr/Hosting/config.json", back, 0777)
						if err != nil {
							return errors.New("cannot write to config file")
						}
						configNuster()
						go exec.Command("/bin/bash", "-c", "service hosting restart").Output()
						return nil
					}
				}
			}
		}
	}

	return errors.New("Domain not found with this app")
}
