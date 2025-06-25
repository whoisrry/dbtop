package monitor

import (
	"log"
	"time"

	"dbtop/config"
	"dbtop/monitor/drivers"
	"dbtop/ui"
)

// Start begins monitoring the specified database instance
func Start(instanceName string, instance config.DatabaseInstance) {
	// Get the appropriate driver for the database type
	driver, err := drivers.GetDriver(instance.Type)
	if err != nil {
		log.Fatal("Failed to get database driver:", err)
	}

	// Connect to the database
	db, err := driver.Connect(instance)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize the UI
	ui := ui.NewUI(instanceName, instance.Type, instance.RefreshInterval)
	defer ui.Close()

	// Start monitoring loop
	ticker := time.NewTicker(instance.RefreshInterval)
	defer ticker.Stop()

	// Channel for UI events
	uiEvents := make(chan string, 10)

	// Start UI event handling in a goroutine
	go func() {
		for {
			select {
			case event := <-uiEvents:
				if !ui.HandleKey(event) {
					return
				}
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			// Get database statistics
			stats, err := driver.GetStats(db, instance.Database)
			if err != nil {
				log.Printf("Failed to get database stats: %v", err)
				continue
			}

			// Update the UI
			ui.Update(stats)
		}
	}
}
