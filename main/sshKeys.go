package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

func addSshKey(c echo.Context) error {
	ssh := new(SSH)
	if err := c.Bind(ssh); err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid request")
	}

	out, err := exec.Command("/bin/bash", "-c", fmt.Sprintf("echo \"%s\" | ssh-keygen -l -f -", ssh.Key)).CombinedOutput()
	if err != nil {
		log.Print(string(out))
		body := new(FieldError)
		body.Error.Field = "key"
		body.Error.Message = "Invalid SSH Key"
		return c.JSON(http.StatusBadRequest, body)
	}

	var authorizedKeysPath string
	if ssh.User != "root" {
		authorizedKeysPath = fmt.Sprintf("/home/%s/.ssh/authorized_keys", ssh.User)
		os.MkdirAll(fmt.Sprintf("/home/%s/.ssh", ssh.User), 0700)
		exec.Command("/bin/bash", "-c", fmt.Sprintf("chown %[1]s:%[1]s -R /home/%[1]s/.ssh", ssh.User)).Output()
	} else {
		authorizedKeysPath = "/root/.ssh/authorized_keys"
		os.MkdirAll("/root/.ssh", 0700)
	}

	f, err := os.OpenFile(authorizedKeysPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}
	defer f.Close()

	// Append the SSH key with associated label and timestamp to the authorized_keys file
	if ssh.Label != "" && ssh.Timestamp != 0 {
		keyLine := fmt.Sprintf("#hosting/%s/%d\n%s\n", ssh.Label, ssh.Timestamp, ssh.Key)
		f.Write([]byte(keyLine))
	} else {
		f.Write([]byte("\n" + ssh.Key))
	}

	return c.NoContent(200)
}

func removeSshKey(c echo.Context) error {
	ssh := new(SSH)
	if err := c.Bind(ssh); err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid request")
	}

	var authorizedKeysPath string
	if ssh.User != "root" {
		authorizedKeysPath = fmt.Sprintf("/home/%s/.ssh/authorized_keys", ssh.User)
	} else {
		authorizedKeysPath = "/root/.ssh/authorized_keys"
	}

	// Read the existing authorized_keys file
	fileContent, err := ioutil.ReadFile(authorizedKeysPath)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Failed to read authorized_keys file")
	}

	// Remove the specified SSH key and its associated comment (if present)
	lines := strings.Split(string(fileContent), "\n")
	var newContent strings.Builder
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "#hosting") && i+1 < len(lines) {
			// Check for comments starting with "#hosting" and extract the label and timestamp
			fields := strings.Split(line, "/")
			if len(fields) >= 3 && fields[0] == "#hosting" {
				_, err := strconv.ParseInt(fields[2], 10, 64)
				if err != nil {
					// Invalid timestamp, skip this comment
					continue
				}
				// Check if the current line contains the specified SSH key
				if strings.Contains(lines[i+1], ssh.Key) {
					// Skip this line (the key) and the next line (the associated comment)
					i++
					continue
				}
				// Append the comment and the key (in case the key is not found)
				newContent.WriteString(line)
				newContent.WriteString("\n")
			}
		} else if !strings.Contains(line, ssh.Key) {
			// Append the line if it doesn't contain the specified key
			newContent.WriteString(line)
			newContent.WriteString("\n")
		}
	}

	// Write the modified content back to the authorized_keys file
	err = ioutil.WriteFile(authorizedKeysPath, []byte(newContent.String()), 0600)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Failed to update authorized_keys file")
	}

	return c.JSON(http.StatusOK, "Success")
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
