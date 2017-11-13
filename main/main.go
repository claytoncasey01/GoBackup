package main

import (
	"RTCWatcher/backup"
	"time"
)

const configPath = "./config.json"

func main() {
	// Load up the config file so we know where to backup to.
	backUp := backup.Backup{}

	// Do the first backup right at program start
	backUp.LoadConfig(configPath)
	backUp.CheckChanged()

	// Then after the initial backup begin doing a backup every X number of hours
	ticker := time.NewTicker(time.Duration(backUp.TimeBetweenBackups) * time.Hour)
	start(ticker, &backUp)

}

func start(ticker *time.Ticker, backUp *backup.Backup) {
	for {
		select {
		case <-ticker.C:
			backUp.LoadConfig(configPath)
			backUp.CheckChanged()
		}
	}
}
