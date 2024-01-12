package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/labstack/echo/v4"
)

func updateipdeny(c echo.Context) error {
	conf := new(struct {
		App string   `json:"app"`
		Ip  []string `json:"ips"`
	})
	c.Bind(&conf)
	if len(conf.Ip) > 0 {
		file, err := os.OpenFile(fmt.Sprintf("/usr/local/lsws/conf/vhosts/%s.d/modules/accessdeny.conf", conf.App), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0750)
		if err != nil {
			return AbortWithErrorMessage(c, "File error")
		}
		ipstring := strings.Join(conf.Ip, ", ")
		file.Write([]byte(fmt.Sprintf(`
accessControl  {
  allow                   *
  deny                    <<<END_deny
		%s
  END_deny

}`, ipstring)))
		file.Close()
		defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
		return c.JSON(200, "Success")
	} else {
		exec.Command("/bin/bash", "-c", fmt.Sprintf("rm /usr/local/lsws/conf/vhosts/%s.d/modules/accessdeny.conf", conf.App)).Output()
		defer exec.Command("/bin/bash", "-c", "service lsws reload").Output()
		return c.JSON(200, "Success")

	}
}
