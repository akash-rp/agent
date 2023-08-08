package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

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

// readKeysFromFile reads authorized SSH keys with comments starting with "#hosting" from a file.
func readKeysFromFile(username, filename string) ([]AuthorizedKey, error) {
	var authorizedKeys []AuthorizedKey

	file, err := os.Open(filename)
	if err != nil {
		// Return an empty list of keys if the file does not exist
		if os.IsNotExist(err) {
			return authorizedKeys, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentLabel string
	var currentTimestamp int64
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip lines that start with '#' but not '#hosting'
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#hosting") {
			continue
		}

		if strings.HasPrefix(line, "#hosting/") {
			// Check for comments starting with "#hosting" and extract label and timestamp
			fields := strings.Split(line, "/")
			if len(fields) >= 3 && fields[0] == "#hosting" {
				currentLabel = fields[1]
				currentTimestamp, err = strconv.ParseInt(fields[2], 10, 64)
				if err != nil {
					// Invalid timestamp, skip this comment
					continue
				}
				continue
			}
		}

		// If the line is not empty, add the authorized key with associated label and timestamp
		if line != "" {
			authorizedKeys = append(authorizedKeys, AuthorizedKey{
				Username:  username,
				Key:       line,
				Label:     currentLabel,
				Timestamp: currentTimestamp,
			})
		}

		// Reset label and timestamp to empty for the next key
		currentLabel = ""
		currentTimestamp = 0
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return authorizedKeys, nil
}

// ReadAllAuthorizedKeys reads all authorized SSH keys with comments for all users on Ubuntu.
func ReadAllAuthorizedKeys() ([]AuthorizedKey, error) {
	var authorizedKeys []AuthorizedKey

	// Read /etc/passwd file
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) != 7 {
			// Invalid line format, skip it
			continue
		}
		username := fields[0]
		homeDir := fields[5]

		// Read the authorized_keys file for the current user
		authorizedKeysFile := filepath.Join(homeDir, ".ssh", "authorized_keys")
		keys, err := readKeysFromFile(username, authorizedKeysFile)
		if err != nil {
			// Skip users with no authorized_keys file or read errors
			continue
		}

		// Append the keys to the list
		authorizedKeys = append(authorizedKeys, keys...)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return authorizedKeys, nil
}

func getSshKeys(c echo.Context) error {
	authorizedKeys, err := ReadAllAuthorizedKeys()
	if err != nil {
		fmt.Println("Error:", err)
		return c.NoContent(400)
	}

	return c.JSON(200, authorizedKeys)
}
