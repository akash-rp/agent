package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func addSSHkey(c echo.Context) error {
	user := c.Param("user")
	ssh := new(SSH)
	c.Bind(&ssh)
	os.MkdirAll(fmt.Sprintf("/home/%s/.ssh", user), 0700)
	f, err := os.OpenFile(fmt.Sprintf("/home/%s/.ssh/authorized_keys", user), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0700)
	exec.Command("/bin/bash", "-c", fmt.Sprintf("chown %[1]s:%[1]s -R /home/%[1]s/.ssh", user)).Output()
	if err != nil {
		return c.JSON(404, "Cannot open file")
	}
	f.Write([]byte("\n" + ssh.Key))
	f.Close()
	return c.JSON(200, "Success")
}

func removeSSHkey(c echo.Context) error {
	user := c.Param("user")
	ssh := new(SSH)
	c.Bind(&ssh)
	_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("grep -v \"%s\" /home/%[2]s/.ssh/authorized_keys > /home/%[2]s/.ssh/tmp; mv -f /home/%[2]s/.ssh/tmp /home/%[2]s/.ssh/authorized_keys; chown %[2]s:%[2]s /home/%[2]s/.ssh/authorized_keys", ssh.Key, user)).CombinedOutput()
	if err != nil {

		return c.JSON(404, "Cannot delete ssh key")
	}
	return c.JSON(200, "success")
}
