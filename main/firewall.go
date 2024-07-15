package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func update7G(c echo.Context) error {
	conf := new(struct {
		App     string   `json:"app"`
		User    string   `json:"user"`
		Enabled bool     `json:"enabled"`
		Disable []string `json:"disable"`
	})
	c.Bind(&conf)
	path := fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/rewrite.conf", conf.App)
	switch conf.Enabled {
	case true:
		exec.Command("/bin/bash", "-c", fmt.Sprintf("cp /usr/Hosting/firewall/7g/rewrite.conf %s", path)).Output()
		exec.Command("/bin/bash", "-c", fmt.Sprintf("cp /usr/Hosting/firewall/7g/7G_log.php /home/%s/%s/public/7G_log.php", conf.User, conf.App)).Output()
		for _, part := range conf.Disable {
			switch part {
			case "query":
				log.Print("Query")
				out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/QUERY STRING/,/END QUERY STRING/d' %s", path)).CombinedOutput()
				if err != nil {
					log.Print(string(out))
					log.Print(err.Error())
					return AbortWithErrorMessage(c, "Error")
				}
			case "request":
				exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/REQUEST URI/,/END REQUEST URI/d' %s", path)).Output()
			case "agent":
				exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/USER AGENT/,/END USER AGENT/d' %s", path)).Output()
			case "host":
				exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/REMOTE HOST/,/END REMOTE HOST/d' %s", path)).Output()
			case "referrer":
				exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/HTTP REFERRER/,/END HTTP REFERRER/d' %s", path)).Output()
			case "method":
				exec.Command("/bin/bash", "-c", fmt.Sprintf("sed -i '/REQUEST METHOD/,/END REQUEST METHOD/d' %s", path)).Output()

			}
		}
		defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
		return c.JSON(200, "Success")
	case false:
		exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /home/%s/%s/public/7g_log.php", conf.User, conf.App)).Output()

		file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/rewrite.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return AbortWithErrorMessage(c, "Unable to open file")
		}
		file.Write([]byte(`
rewrite {
	enable 			 1
	autoLoadHtaccess 1
}`))
		file.Close()
		defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()

		return c.JSON(200, "Success")
	}
	return AbortWithErrorMessage(c, "Something went wrong")
}

func updateModsecurity(c echo.Context) error {
	conf := new(struct {
		App              string `json:"app"`
		Enabled          bool   `json:"enabled"`
		ParanoiaLevel    int    `json:"paranoiaLevel"`
		AnomalyThreshold int    `json:"anomalyThreshold"`
	})
	c.Bind(&conf)

	switch conf.Enabled {
	case true:
		if conf.ParanoiaLevel <= 0 && conf.ParanoiaLevel > 4 && conf.AnomalyThreshold < 5 && conf.AnomalyThreshold > 100 {
			return AbortWithErrorMessage(c, "Invalid levels")
		}
		err := os.MkdirAll(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/modsecurity.d/", conf.App), 0750)
		if err != nil {
			return AbortWithErrorMessage(c, "Error creating folder")
		}
		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("cp -r /usr/Hosting/firewall/coreruleset /usr/local/lsws/conf/vhosts/%s.d/modules/modsecurity.d/", conf.App)).CombinedOutput()
		if err != nil {
			log.Print(out)
			return AbortWithErrorMessage(c, "Error copying coreruleset")
		}
		file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/modsecurity.d/main.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return AbortWithErrorMessage(c, "Error writing main file")
		}
		file.Write([]byte(fmt.Sprintf(`
include /usr/local/lsws/conf/vhosts/%[1]s.d/modules/modsecurity.d/coreruleset/crs-setup.conf
include /usr/local/lsws/conf/vhosts/%[1]s.d/modules/modsecurity.d/coreruleset/rules/*.conf
`, conf.App)))
		file.Close()
		modsec, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/modsecurity.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return AbortWithErrorMessage(c, "Error writing Modsec file")
		}
		modsec.Write([]byte(fmt.Sprintf(`
module mod_security {
	modsecurity					On
	modsecurity_rules   		`+"`"+`
	SecRuleEngine				On
	SecAuditLogParts			ABCDEFGHJK
	SecAuditEngine 				RelevantOnly
	SecAuditLog 				/var/log/hosting/%[1]s/modsec.log
	SecAuditLogType 			Serial
	SecAuditLogRelevantStatus 	"^(?:5|4(?!04))"
	SecAction "id:900000,phase:1,nolog,pass,t:none,setvar:tx.paranoia_level=%[2]d"
	SecAction "id:900110,phase:1,nolog,pass,t:none,setvar:tx.inbound_anomaly_score_threshold=%[3]d,setvar:tx.outbound_anomaly_score_threshold=%[3]d"
	`+"`"+`
	modsecurity_rules_file          /usr/local/lsws/conf/vhosts/%[1]s.d/modules/modsecurity.d/main.conf
  	ls_enabled              1
 }
`, conf.App, conf.ParanoiaLevel, conf.AnomalyThreshold)))
		modsec.Close()
		defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
		return c.JSON(200, "Success")
	case false:
		exec.Command("/bin/bash", "-c", fmt.Sprintf("rm -rf /usr/local/lsws/conf/vhosts/%[1]s.d/modules/modsecurity.*", conf.App)).Output()
		defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()

		return c.JSON(200, "Success")
	}
	return AbortWithErrorMessage(c, "Invalid Request")
}
