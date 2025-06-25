package drivers

import (
	"database/sql"
	"fmt"

	"dbtop/config"
	"dbtop/monitor/stats"
)

// Driver defines the interface for database drivers
type Driver interface {
	Connect(instance config.DatabaseInstance) (*sql.DB, error)
	GetStats(db *sql.DB, database string) (*stats.DatabaseStats, error)
}

var drivers = make(map[string]Driver)

// RegisterDriver registers a new database driver
func RegisterDriver(name string, driver Driver) {
	drivers[name] = driver
}

// GetDriver returns the driver for the specified database type
func GetDriver(dbType string) (Driver, error) {
	driver, exists := drivers[dbType]
	if !exists {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
	return driver, nil
}

// GetSupportedDrivers returns a list of supported database types
func GetSupportedDrivers() []string {
	var supported []string
	for driver := range drivers {
		supported = append(supported, driver)
	}
	return supported
}
