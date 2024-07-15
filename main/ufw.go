package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sort"

	"github.com/labstack/echo/v4"
)

func getUfwRules(c echo.Context) error {
	out, err := exec.Command("/bin/bash", "-c", "ufw status numbered | jc --ufw").CombinedOutput()
	type Ufw struct {
		Status string
		Rules  []struct {
			Index            int
			Action           string
			Network_protocol string
			To_ports         []int
			To_transport     string
			From_ip          string
			To_port_ranges   []struct {
				Start int
				End   int
			}
		}
	}
	if err != nil {
		log.Print("First")
		log.Print(string(out))
		return AbortWithErrorMessage(c, err.Error())
	}
	log.Print(string(out))
	ufw := new(Ufw)
	err = json.Unmarshal(out, &ufw)
	if err != nil {
		log.Print("Second")

		return AbortWithErrorMessage(c, err.Error())
	}
	return c.JSON(200, ufw)
}

func deleteUfwRules(c echo.Context) error {
	type Ufw struct {
		Index []int `json:"index"`
	}
	ufw := new(Ufw)
	c.Bind(&ufw)
	sort.Slice(ufw.Index, func(i, j int) bool {
		return ufw.Index[i] > ufw.Index[j]
	})
	for _, index := range ufw.Index {
		exec.Command("/bin/bash", "-c", fmt.Sprintf("ufw --force delete %d", index)).CombinedOutput()
	}
	return getUfwRules(c)
}

type addRules struct {
	Source struct {
		Type   string `json:"type"`
		Ip     string `json:"ip"`
		Subnet struct {
			Ip     string `json:"ip"`
			Prefix string `json:"prefix"`
		} `json:"subnet"`
	} `json:"source"`
	Port struct {
		Type   string   `json:"type"`
		Number string   `json:"number"`
		Range  []string `json:"range"`
	} `json:"port"`
	Protocol string `json:"protocol"`
	Action   string `json:"action"`
}

func addUfwRules(c echo.Context) error {

	rule := new(addRules)
	err := c.Bind(&rule)
	if err != nil {
		log.Print(err.Error())
		return c.NoContent(400)
	}

	log.Print("genetate ufw")
	add := generateUfwRule(*rule)
	log.Print("genetated ufw wefwefw")
	log.Print(add)

	_, err = exec.Command("/bin/bash", "-c", add).Output()
	if err != nil {
		log.Print(add)
		log.Print(err.Error())
		return c.NoContent(400)
	}
	return getUfwRules(c)
}

func generateUfwRule(rule addRules) string {
	add := "ufw "
	return generateAction(rule, add)
}

func generateAction(rule addRules, add string) string {
	if rule.Action == "allow" {
		add = add + "allow "
	} else {
		add = add + "reject "
	}
	return generateFrom(rule, add)
}

func generateFrom(rule addRules, add string) string {
	switch rule.Source.Type {
	case "any":
		add = add + "from any "
	case "ipv4":
		add = add + "from 0.0.0.0/0 "
	case "ipv6":
		add = add + "from ::/0 "
	case "single":
		add = add + fmt.Sprintf("from %s ", rule.Source.Ip)
	case "subnet":
		add = add + fmt.Sprintf("from %s/%s ", rule.Source.Subnet.Ip, rule.Source.Subnet.Prefix)
	}
	return generatePort(rule, add)
}

func generatePort(rule addRules, add string) string {
	switch rule.Port.Type {
	case "any":
		add = add + "to any "
	case "single":
		add = add + "to any port " + rule.Port.Number + " "
	case "range":
		add = add + "to any port " + rule.Port.Range[0] + ":" + rule.Port.Range[1] + " "
	}
	return generateProto(rule, add)
}

func generateProto(rule addRules, add string) string {
	switch rule.Protocol {
	case "all":
		add = add + "proto any"
	case "tcp":
		add = add + "proto tcp"
	case "udp":
		add = add + "proto udp"
	}
	return add
}
