package main

import (
	"fmt"
	"os/exec"
	"time"
)

func initCron() {
	fmt.Print(obj)
	for _, site := range obj.Sites {
		if (site.LocalBackup != Backup{}) {
			switch site.LocalBackup.Frequency {
			case "1 Hour":
				_, err := cronInt.Cron(fmt.Sprintf("%d * * * *", site.LocalBackup.Minute)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akash")
				})
				if err != nil {
					fmt.Println(err)
				}

			case "2 Hours":
				cronInt.Cron(fmt.Sprintf("%d */2 * * *", site.LocalBackup.Minute)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akash")
				})

			case "3 Hours":
				cronInt.Cron(fmt.Sprintf("%d */3 * * *", site.LocalBackup.Minute)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akashprint")
				})
			case "6 Hours":
				cronInt.Cron(fmt.Sprintf("%d */6 * * *", site.LocalBackup.Minute)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akash")
				})
			case "12 Hours":
				cronInt.Cron(fmt.Sprintf("%d */12 * * *", site.LocalBackup.Minute)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akash")
				})
			case "1 Day":
				cronInt.Every(1).Day().At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akash")
				})
			case "3 Days":
				cronInt.Every(3).Day().At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akash")
				})
			case "1 Week":
				switch site.LocalBackup.WeekDay {
				case "Sunday":
					cronInt.Every(1).Weekday(time.Sunday).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
						fmt.Printf("akash")
					})
				case "Monday":
					cronInt.Every(1).Weekday(time.Monday).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
						fmt.Printf("akash")
					})
				case "Tuesday":
					cronInt.Every(1).Weekday(time.Tuesday).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
						fmt.Printf("akash")
					})
				case "Wednesday":
					cronInt.Every(1).Weekday(time.Wednesday).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
						fmt.Printf("akash")
					})
				case "Thursday":
					cronInt.Every(1).Weekday(time.Thursday).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
						fmt.Printf("akash")
					})
				case "Friday":
					cronInt.Every(1).Weekday(time.Friday).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
						fmt.Printf("akash")
					})
				case "Saturday":
					cronInt.Every(1).Weekday(time.Saturday).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
						fmt.Printf("akash")
					})
				}

			case "1 Month":
				day := site.LocalBackup.MonthDay
				cronInt.Every(1).Month(day).At(fmt.Sprintf("%s", site.LocalBackup.Time)).Tag(fmt.Sprintf("%s", site.Name)).Do(func() {
					fmt.Printf("akash")
				})
			}

		}
	}
	cronInt.StartAsync()
	fmt.Println("AKASH")
	fmt.Println(cronInt.NextRun())
}

func takeBackup() {

}

func addNewBackup(name string, user string) {

	exec.Command(fmt.Sprintf("kopia repository create filesystem --path='/var/Backup/%s' --password=%s ; kopia policy set --keep-latest %s --global; kopia snapshot create /home/%s/%s ", name, name, user, name)).Output()

}
