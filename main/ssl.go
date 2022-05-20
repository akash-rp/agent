package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func addSslCert(c echo.Context) error {
	conf := new(sslConf)
	c.Bind(&conf)
	switch conf.SslMethod {
	case "webroot":
		err := webroot(*conf, "new")
		if err != nil {
			return c.JSON(400, err.Error())
		}
		return c.JSON(200, "Success")

	}
	return c.JSON(400, "Something went wrong")
}

func reissueSslCert(c echo.Context) error {
	conf := new(sslConf)
	c.Bind(&conf)
	switch conf.SslMethod {
	case "webroot":
		err := webroot(*conf, "reissue")
		if err != nil {
			return c.JSON(400, err.Error())
		}
		return c.JSON(200, "success")
	}
	return c.JSON(400, "Something went wrong")
}

func resolveDomain(conf sslConf) (Domain string, FolderName string) {
	id := uuid.New()
	Domain = ""
	FolderName = ""
	FolderSet := false
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	ioutil.WriteFile(fmt.Sprintf("/home/%s/%s/public/.sslresolve", conf.User, conf.App), []byte(id.String()), 0744)
	res, err := client.Get(fmt.Sprintf("http://%s/.sslresolve", conf.Domain))
	if err == nil {
		FolderName = conf.Domain
		FolderSet = true
		body, _ := ioutil.ReadAll(res.Body)
		if string(body) == id.String() {
			Domain = "-d " + conf.Domain
		}
	}
	time.Sleep(1 * time.Second)
	id = uuid.New()
	ioutil.WriteFile(fmt.Sprintf("/home/%s/%s/public/.sslresolve", conf.User, conf.App), []byte(id.String()), 0744)
	res, err = client.Get(fmt.Sprintf("http://www.%s/.sslresolve", conf.Domain))
	if err != nil {
		return Domain, FolderName
	}
	body, _ := ioutil.ReadAll(res.Body)
	if string(body) == id.String() {
		if !FolderSet {
			FolderName = "www." + conf.Domain
		}
		Domain = Domain + " -d " + "www." + conf.Domain
	}
	return Domain, FolderName
}

func configureDomainForSSl(conf sslConf, FolderName string) {
	file, _ := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/domain/%s.conf", conf.App, conf.Domain), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
	file.Write([]byte(fmt.Sprintf(`
# Editing this file manually might change litespeed behavior,
# Make sure you know what are you doing
virtualhost %[1]s {
  listeners http, https

  vhDomain                  %[1]s

  rewrite  {
    enable                  1
    autoLoadHtaccess        1

    RewriteCond %%{HTTP:CF-Visitor} '"scheme":"http"' [OR]
    RewriteCond %%{HTTPS} !=on
    RewriteRule ^(.*)$ - [env=proto:http]
    RewriteCond %%{HTTP:CF-Visitor} '"scheme":"https"' [OR]
    RewriteCond %%{HTTPS} =on
    RewriteRule ^(.*)$ - [env=proto:https]

    # Redirect http -> https
    RewriteCond %%{HTTPS} off
    RewriteRule (.*) https://%[1]s%%{REQUEST_URI} [R=301,L]

  }

  vhssl {
    keyFile                 /usr/local/lsws/certs/%[2]s/%[2]s.key
    certFile                /usr/local/lsws/certs/%[2]s/%[2]s.key
    certChain               1
    enableECDHE             1
    enableStapling          1
    ocspRespMaxAge          86400
  }

  include /usr/local/lsws/conf/vhosts/%[3]s.d/main.conf
}`, conf.Domain, FolderName, conf.App)))
	file.Close()
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
}

func webroot(conf sslConf, procedure string) error {
	ExisitingFolderName := ""
	if procedure == "reissue" {
		for _, site := range obj.Sites {
			if site.Name == conf.App {
				if site.PrimaryDomain.Url == conf.Domain {
					ExisitingFolderName = site.PrimaryDomain.SSL.FolderName
					break
				} else {
					for _, alias := range site.AliasDomain {
						if alias.Url == conf.Domain {
							ExisitingFolderName = alias.SSL.FolderName
							break
						}
					}
				}
			}
		}
		if ExisitingFolderName == "" {
			return errors.New("reissue needs existing ssl")
		}
	}
	var domainFinal string
	var FolderName string
	if conf.IsSubdomain {
		domainFinal = conf.Domain
		FolderName = conf.Domain
	} else {

		domainFinal, FolderName = resolveDomain(conf)
		log.Print(domainFinal, FolderName)
	}
	if domainFinal == "" {
		return errors.New("no domain resolved")
	}
	_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("/root/.acme.sh/acme.sh --issue %s -w /home/%s/%s/public --test --force", domainFinal, conf.User, conf.App)).Output()
	if err != nil {
		return errors.New("SSL failed to staging")
	}
	_, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("/root/.acme.sh/acme.sh --issue %s -w /home/%s/%s/public --force", domainFinal, conf.User, conf.App)).Output()
	if err != nil {
		return errors.New("SSL failed")
	}
	if procedure == "new" || FolderName != ExisitingFolderName {
		log.Print("Entered the config change loop")
		for i, site := range obj.Sites {
			if site.Name == conf.App {
				if site.PrimaryDomain.Url == conf.Domain {
					site.PrimaryDomain.SSL.FolderName = FolderName
					obj.Sites[i].PrimaryDomain = site.PrimaryDomain
					break
				} else {
					for j, alias := range site.AliasDomain {
						alias.SSL.FolderName = FolderName
						obj.Sites[i].AliasDomain[j] = alias
					}
				}
			}
		}
	}

	defer SaveJSONFile()
	if procedure == "new" {
		configureDomainForSSl(conf, FolderName)
	} else if FolderName != ExisitingFolderName {
		log.Print("Changing lines now using sed")
		exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/keyFile/c\\    keyFile				/usr/local/lsws/certs/%[1]s/%[1]s.key' /usr/local/lsws/conf/vhosts/%[2]s.d/domain/%[1]s.conf", FolderName, conf.App)).Output()
		exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/certFile/c\\   certFile				/usr/local/lsws/certs/%[1]s/fullchair.cer' /usr/local/lsws/conf/vhosts/%[2]s.d/domain/%[1]s.conf", FolderName, conf.App)).Output()
		defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
	}
	//Need to get old folder name and if folder changes then only change two lines to cert files
	return nil
}

// func wildcard(conf sslConf, procedure string) error {
// 	ExisitingFolderName := ""
// 	if procedure == "reissue" {
// 		for _, site := range obj.Sites {
// 			if site.Name == conf.App {
// 				if site.PrimaryDomain.Url == conf.Domain {
// 					ExisitingFolderName = site.PrimaryDomain.SSL.FolderName
// 					break
// 				} else {
// 					for _, alias := range site.AliasDomain {
// 						if alias.Url == conf.Domain {
// 							ExisitingFolderName = alias.SSL.FolderName
// 							break
// 						}
// 					}
// 				}
// 			}
// 		}
// 		if ExisitingFolderName == "" {
// 			return errors.New("reissue needs existing ssl")
// 		}
// 	}
// 	domainFinal := "-d " + conf.Domain + " -d *." + conf.Domain
// 	FolderName := conf.Domain

// }
