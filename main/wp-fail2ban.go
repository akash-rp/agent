package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/labstack/echo/v4"
)

//Kept for pending. cannot use now
func enableWpFail2Ban(c echo.Context) error {
	type site struct {
		Name     string   `json:"name"`
		User     string   `json:"user"`
		Disabled []string `json:"disabled"`
	}
	body := new(site)
	c.Bind(&body)
	_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin is-installed wp-fail2ban --path='/home/%[1]s/%[2]s/public'", body.User, body.Name)).CombinedOutput()
	if err != nil {

		out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("sudo -u %[1]s -i /usr/Hosting/wp-cli plugin install wp-fail2ban --path='/home/%[1]s/%[2]s/public'", body.User, body.Name)).CombinedOutput()
		if err != nil {
			log.Print(string(out))
			return c.NoContent(400)
		}
	}
	out, err := exec.Command("bin/bash", "-c", fmt.Sprintf("mkdir -p /home/%[1]s/%[2]s/public/wp-content/mu-plugins ; chmod 755 /home/%[1]s/%[2]s/public/wp-content/mu-plugins; chown %[1]s:%[1]s /home/%[1]s/%[2]s/public/wp-content/mu-plugins", body.User, body.Name)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		return c.NoContent(400)
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("ln -s /home/%[1]s/%[2]s/public/wp-content/plugins/wp-fail2ban/wp-fail2ban.php /home/%[1]s/%[2]s/public/wp-content/mu-plugins/wp-fail2ban.php", body.User, body.Name)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		return c.NoContent(400)
	}
	out, err = exec.Command("/bin/bash", "-c", fmt.Sprintf("awk '/wp-settings.php/{print \"include ABSPATH . 'wp-fail2ban-config.php';\\n\"}1' /home/%s/%s/public/wp-config.php", body.User, body.Name)).Output()
	if err != nil {
		log.Print(string(out))
		return c.NoContent(400)
	}
	return nil
}
