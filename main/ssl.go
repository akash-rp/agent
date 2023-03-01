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
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func addSslCert(c echo.Context) error {
	conf := new(sslConf)
	c.Bind(&conf)
	switch conf.Challenge {
	case "http-01":
		err := webroot(*conf)
		if err != nil {
			return c.JSON(400, err.Error())
		}
		return c.JSON(200, "Success")
	case "dns-01":
		err := dnsApi(*conf)
		if err != nil {
			return c.JSON(400, err.Error())
		}
		return c.JSON(200, "success")

	}
	return c.JSON(400, "Something went wrong")
}

// func reissueSslCert(c echo.Context) error {
// 	conf := new(sslConf)
// 	c.Bind(&conf)
// 	switch conf.SslMethod {
// 	case "webroot":
// 		err := webroot(*conf)
// 		if err != nil {
// 			return c.JSON(400, err.Error())
// 		}
// 		return c.JSON(200, "success")
// 	}
// 	return c.JSON(400, "Something went wrong")
// }

func resolveDomain(conf sslConf) error {

	if len(conf.Domains) == 0 {
		return errors.New("no domains provided")
	}
	for _, domain := range conf.Domains {
		if strings.Contains(domain, "*.") {
			domain = "test." + conf.Domain
		}
		id := uuid.New()
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		ioutil.WriteFile(fmt.Sprintf("/home/%s/%s/public/.sslresolve", conf.User, conf.App), []byte(id.String()), 0744)
		res, err := client.Get(fmt.Sprintf("http://%s/.sslresolve", domain))
		if err != nil {
			log.Print(err.Error())
			log.Print(domain)
			return errors.New("domains not resolved")
		}
		body, _ := ioutil.ReadAll(res.Body)
		if string(body) != id.String() {
			return errors.New("domains Not resolves")
		}
		linuxCommand(fmt.Sprintf("rm /home/%s/%s/public/.sslresolve", conf.User, conf.App))
		time.Sleep(1 * time.Second)
	}
	return nil
}

func configureDomainForSSl(AppName string, Domain string) {
	file, _ := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/domain/%s.ssl", AppName, Domain), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)

	// virtualhost %[1]s {
	// 	listeners http, https

	// 	vhDomain                  %[1]s

	// 	rewrite  {
	// 	  enable                  1
	// 	  autoLoadHtaccess        1

	// 	  RewriteCond %%{HTTP:CF-Visitor} '"scheme":"http"' [OR]
	// 	  RewriteCond %%{HTTPS} !=on
	// 	  RewriteRule ^(.*)$ - [env=proto:http]
	// 	  RewriteCond %%{HTTP:CF-Visitor} '"scheme":"https"' [OR]
	// 	  RewriteCond %%{HTTPS} =on
	// 	  RewriteRule ^(.*)$ - [env=proto:https]

	//   # Redirect http -> https
	//   RewriteCond %%{HTTPS} off
	//   RewriteRule (.*) https://%%{HTTP_HOST}%%{REQUEST_URI} [R=301,L]

	// 	}
	// 	include /usr/local/lsws/conf/vhosts/%[3]s.d/main.conf
	// }

	file.Write([]byte(fmt.Sprintf(`
# Editing this file manually might change litespeed behavior,
# Make sure you know what are you doing
  vhssl {
    keyFile                 /etc/letsencrypt/live/%[1]s/privkey.pem
    certFile                /etc/letsencrypt/live/%[1]s/fullchain.pem
    certChain               1
    enableECDHE             1
    enableStapling          1
    ocspRespMaxAge          86400
  }

 `, Domain)))
	file.Close()
	defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
}

func webroot(conf sslConf) error {
	err := resolveDomain(conf)
	if err != nil {
		linuxCommand(fmt.Sprintf("rm /home/%s/%s/public/.sslresolve", conf.User, conf.App))
		return err
	}
	out, err := linuxCommand(fmt.Sprintf(" certbot certonly --cert-name %[1]s -d %[2]s --agree-tos --webroot -w /home/%[3]s/%[4]s/public/ -n --email akashrp@outlook.com --force-renewal --dry-run", conf.Domain, strings.Join(conf.Domains, ","), conf.User, conf.App))
	if err != nil {
		log.Print(string(out))
		return errors.New("dry Run Failed")
	}
	provider := ""
	if conf.Provider == "Zerossl" {
		provider = "--eab-kid sGecNMFE7aXC7HSG12j12g --eab-hmac-key 6Js6yb2xl0Km3KMLekm7YRP974gpcbCheHkIAMWig6BPt8RGisjuiSgh88aULztjFJaf8PzPnppkdoiiB6tMqA --server https://acme.zerossl.com/v2/DV90"
	}
	out, err = linuxCommand(fmt.Sprintf(" certbot certonly --cert-name %[1]s -d %[2]s --agree-tos --webroot -w /home/%[3]s/%[4]s/public/ -n --email akashrp@outlook.com --force-renewal %[5]s", conf.Domain, strings.Join(conf.Domains, ","), conf.User, conf.App, provider))
	if err != nil {
		log.Print(string(out))
		return errors.New("failed to issue certificate")
	}
	configureDomainForSSl(conf.App, conf.Domain)
	return nil
}

func dnsApi(conf sslConf) error {

	dns := ""
	if conf.DNSProvider == "Cloudflare" {
		confPath := fmt.Sprintf("/usr/Hosting/dns/%s-%s", conf.DNSProvider, conf.Domain)
		err := ioutil.WriteFile(confPath, []byte(fmt.Sprintf("dns_cloudflare_api_token = %s", conf.Token)), 0600)
		if err != nil {
			return err
		}
		dns = fmt.Sprintf("--dns-cloudflare --dns-cloudflare-credentials %s --dns-cloudflare-propagation-seconds 30", confPath)
	} else if conf.DNSProvider == "Digitalocean" {
		confPath := fmt.Sprintf("/usr/Hosting/dns/%s-%s", conf.DNSProvider, conf.Domain)
		err := ioutil.WriteFile(confPath, []byte(fmt.Sprintf("dns_digitalocean_token = %s", conf.Token)), 0600)
		if err != nil {
			return err
		}
		dns = fmt.Sprintf("--dns-digitalocean --dns-digitalocean-credentials %s --dns-digitalocean-propagation-seconds 30", confPath)
	}
	out, err := linuxCommand(fmt.Sprintf("certbot certonly --cert-name %[1]s -d %[2]s %[3]s --agree-tos -n --email akashrp@outlook.com --force-renewal --dry-run ", conf.Domain, strings.Join(conf.Domains, ","), dns))
	if err != nil {
		log.Print(string(out))
		return errors.New("dry Run Failed")
	}
	provider := ""
	if conf.Provider == "Zerossl" {
		provider = "--eab-kid sGecNMFE7aXC7HSG12j12g --eab-hmac-key 6Js6yb2xl0Km3KMLekm7YRP974gpcbCheHkIAMWig6BPt8RGisjuiSgh88aULztjFJaf8PzPnppkdoiiB6tMqA --server https://acme.zerossl.com/v2/DV90"
	}
	out, err = linuxCommand(fmt.Sprintf("certbot certonly --cert-name %[1]s -d %[2]s %[3]s --agree-tos  -n --email akashrp@outlook.com --force-renewal %[4]s", conf.Domain, strings.Join(conf.Domains, ","), dns, provider))
	if err != nil {
		log.Print(string(out))
		return errors.New("failed to issue certificate")
	}
	configureDomainForSSl(conf.App, conf.Domain)
	return nil
}

func listCertificates(c echo.Context) error {
	name := c.Param("name")
	domains := []string{}
	for _, site := range obj.Sites {
		if site.Name == name {
			domains = site.Domains
			break
		}
	}

	certbotOut, err := linuxCommand("certbot certificates")
	if err != nil {
		return c.NoContent(400)
	}

	certSplit := strings.Split(string(certbotOut), "Certificate Name:")
	type Cert struct {
		Name    string   `json:"name"`
		Domains []string `json:"domains"`
		Expiry  string   `json:"expiry"`
	}
	certs := []Cert{}
	for i, cert := range certSplit {
		if i == 0 {
			continue
		}
		certLine := strings.Split(cert, "\n")
		certificate := Cert{}
		for j, line := range certLine {
			if j == 0 {
				if !contains(domains, strings.TrimSpace(line)) {
					break
				}
				certificate.Name = strings.TrimSpace(line)
				continue
			}
			splited := strings.Split(line, ":")
			for k, each := range splited {
				if k == 0 {
					switch strings.TrimSpace(each) {
					case "Domains":
						certificate.Domains = strings.Split(strings.TrimSpace(splited[1]), " ")
					case "Expiry Date":
						certificate.Expiry = strings.Join(splited[1:], ":")
					}
				}
			}
		}
		// log.Print(certLine[0])
		if len(certificate.Name) > 0 {

			certs = append(certs, certificate)
		}
	}
	// certsJson, _ := json.Marshal(certs)
	return c.JSON(200, certs)
}

func checkIfSslExistsForDomain(domain Domain) bool {
	if _, err := os.Stat(fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", domain.Url)); err == nil {
		if _, err := os.Stat(fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", domain.Url)); err == nil {
			return true
		} else if errors.Is(err, os.ErrNotExist) {
			return false
		}

	} else if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}
