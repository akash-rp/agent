package main

import (
	"fmt"
	"log"

	"github.com/go-co-op/gocron"
)

func initCron() {
	cronInt.SetMaxConcurrentJobs(1, gocron.WaitMode)
	fmt.Print(obj)
	log.Print("Initializing CronJob")
	cronInt.StartAsync()
	for _, site := range obj.Sites {
		addCronJob(site.LocalBackup, site.Name, site.User, site.LocalBackup.LastRun)
		for _, remote := range site.RemoteBackup {
			addRemoteCronJob(remote, site.Name, site.User, remote.LastRun)
		}
	}
	cronInt.Every(1).Minute().Do(storeMetrics)
	cronInt.Every(1).Hour().WaitForSchedule().Do(metricsPartation)
}
