package main

import (
	"fmt"
	"io/ioutil"

	"github.com/labstack/echo/v4"
)

func configNuster() error {

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
	// 	return echo.NewHTTPError(400, err)
	// }

	conf := "##############################\n# Do not edit this file#\n#############################\n"

	conf = conf + fmt.Sprintf(`
global
	nuster cache off data-size %dm
	master-worker
	maxconn %d
	tune.ssl.default-dh-param 2048
	ssl-dh-param-file /opt/Hosting/dhparam.pem
	user hosting
	group hosting
	chroot /var/lib/hosting
	log /dev/log/ local0`, obj.Global.Datasize, obj.Global.Maxconn)

	conf = conf + `
defaults
	log /dev/log/ local0
	option httplog
	mode http
`
	conf = conf + ("\tlog-format \"%[capture.req.hdr(0)] %[capture.res.hdr(0)] %ci [%tr] %b %{+Q}r %TR/%Tw/%Tc/%Tr/%Ta %ST %B %tsc %ac/%fc/%bc/%sc/%rc [%[capture.req.hdr(1)]]\"")
	conf = conf + fmt.Sprintf(`
	option http-ignore-probes
	timeout connect %ds
	timeout client %ds
	timeout server %ds
	timeout http-request 30s`, obj.Default.Timeout.Connect, obj.Default.Timeout.Client, obj.Default.Timeout.Server)

	if len(obj.Sites) == 0 {
		conf = conf + `
	http-errors myerrors
    errorfile 503 /usr/Hosting/errors/404.http`
	}

	conf = conf + `
frontend nonssl
	bind *:80
	option forwardfor
	acl letsencrypt-acl path_beg /.well-known/acme-challenge/
    use_backend letsencrypt-backend if letsencrypt-acl`

	if obj.SSL {
		conf = conf + `
	bind *:443 ssl crt /opt/Hosting/certs/ alpn h2,http/1.1
	http-request set-header X-Forwarded-Proto https if { ssl_fc }`
	}

	if len(obj.Sites) == 0 {
		conf = conf + `
	errorfiles myerrors
    http-response return status 404 default-errorfiles`
	}

	conf = conf + `
	http-request capture req.hdr(Host) len 100
	http-request capture req.fhdr(User-Agent) len 100
	declare capture response len 20
	http-response capture res.hdr(x-cache) id 0
	acl has_domain hdr(Host),map_str(/opt/Hosting/routes.map) -m found
	acl has_wildcard hdr(Host),map_sub(/opt/Hosting/wildcardroutes.map) -m found
	http-request reject if !has_domain !has_wildcard
	acl has_cookie hdr_sub(cookie) wordpress_logged_in
	acl has_path path_sub wp-admin || wp-login
	acl static_file path_end .js || .css || .png || .jpg || .jpeg || .gif || .ico`

	conf = conf + `
	use_backend nocache if has_path || has_cookie
	use_backend static if static_file`
	if len(obj.Sites) != 0 {
		conf = conf + `
	use_backend %[req.hdr(host),map(/opt/Hosting/routes.map)] if { req.hdr(host),map(/opt/Hosting/routes.map) -m found }
	use_backend %[req.hdr(host),map_sub(/opt/Hosting/wildcardroutes.map)] if { req.hdr(host),map_sub(/opt/Hosting/wildcardroutes.map) -m found }`
	}
	for i, backend := range obj.Sites {
		if backend.Type == "live" {
			conf = conf + fmt.Sprintf(`
backend %s
    nuster cache %s
    nuster rule r%d
    http-response set-header x-cache HIT if { nuster.cache.hit }
    http-response set-header x-cache MISS unless { nuster.cache.hit }
    server s1 0.0.0.0:8088`, backend.Name, backend.Cache, i)
		}
	}
	conf = conf + `
backend nocache
    http-response set-header x-cache BYPASS
    server s2 0.0.0.0:8088
backend static
    http-response set-header x-type STATIC
    server s2 0.0.0.0:8088
backend Staging
    http-response set-header x-cache BYPASS
    http-response set-header x-type STAGING
	server s2 0.0.0.0:8088
backend letsencrypt-backend
    server letsencrypt 127.0.0.1:8888`

	conf = conf + "\n"
	// the WriteFile method returns an error if unsuccessful
	err := ioutil.WriteFile("/opt/Hosting/hosting.cfg", []byte(conf), 0777)
	wildSite := "###################### DO NOT EDIT THIS FILE #######################\n"
	appendSite := "##################### DO NOT EDIT THIS FILE #########################\n"
	for _, site := range obj.Sites {
		if site.Type == "live" {

			appendSite = appendSite + fmt.Sprintf("%s %s \n", site.PrimaryDomain.Url, site.Name)

			if site.PrimaryDomain.WildCard {
				wildSite = wildSite + fmt.Sprintf(".%s %s \n", site.PrimaryDomain.Url, site.Name)
			}

			for _, alias := range site.AliasDomain {
				appendSite = appendSite + fmt.Sprintf("%s %s \n", alias.Url, site.Name)

				if alias.WildCard {
					wildSite = wildSite + fmt.Sprintf(".%s %s \n", alias.Url, site.Name)
				}

			}
		} else if site.Type == "staging" {
			appendSite = appendSite + fmt.Sprintf("%s Staging \n", site.PrimaryDomain.Url)
			for _, alias := range site.AliasDomain {
				appendSite = appendSite + fmt.Sprintf("%s Staging \n", alias.Url)

				if alias.WildCard {
					wildSite = wildSite + fmt.Sprintf(".%s Staging \n", alias.Url)
				}

			}
		}
	}
	ioutil.WriteFile("/opt/Hosting/routes.map", []byte(appendSite), 0777)
	ioutil.WriteFile("/opt/Hosting/wildcardroutes.map", []byte(wildSite), 0777)
	// handle this error
	if err != nil {
		// print it out
		return echo.NewHTTPError(404, err)
	}

	return nil

}
