package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/labstack/echo/v4"
)

func addSSHkey(c echo.Context) error {
	ssh := new(SSH)
	c.Bind(&ssh)
	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("echo \"%s\" | ssh-keygen -l -f -", ssh.Key)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		body := new(FieldError)
		body.Error.Field = "key"
		body.Error.Message = "Invalid SSH Key"
		return c.JSON(400, body)

	}
	if ssh.User != "root" {

		os.MkdirAll(fmt.Sprintf("/home/%s/.ssh", ssh.User), 0700)
		f, err := os.OpenFile(fmt.Sprintf("/home/%s/.ssh/authorized_keys", ssh.User), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
		exec.Command("/bin/bash", "-c", fmt.Sprintf("chown %[1]s:%[1]s -R /home/%[1]s/.ssh", ssh.User)).Output()
		if err != nil {
			return c.NoContent(400)
		}
		f.Write([]byte("\n" + ssh.Key))
		f.Close()
	} else {
		os.MkdirAll("/root/.ssh", 0700)
		f, err := os.OpenFile("/root/.ssh/authorized_keys", os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
		if err != nil {
			return c.NoContent(400)
		}
		f.Write([]byte("\n" + ssh.Key))
		f.Close()
	}
	return c.JSON(200, "Success")
}

func removeSSHkey(c echo.Context) error {
	ssh := new(SSH)
	c.Bind(&ssh)
	if ssh.User != "root" {

		_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("grep -v \"%s\" /home/%[2]s/.ssh/authorized_keys > /home/%[2]s/.ssh/tmp; mv -f /home/%[2]s/.ssh/tmp /home/%[2]s/.ssh/authorized_keys; chown %[2]s:%[2]s /home/%[2]s/.ssh/authorized_keys", ssh.Key, ssh.User)).CombinedOutput()
		if err != nil {
			return c.NoContent(400)
		}
	} else {
		_, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("grep -v \"%s\" /root/.ssh/authorized_keys > /root/.ssh/tmp; mv -f /root/.ssh/tmp /root/.ssh/authorized_keys;", ssh.Key)).CombinedOutput()
		if err != nil {
			return c.NoContent(400)
		}
	}
	return c.JSON(200, "success")
}
