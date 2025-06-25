package drivers

import (
	"database/sql"
	"fmt"
	"time"

	"dbtop/config"
	"dbtop/monitor/stats"

	_ "github.com/go-sql-driver/mysql"
)

type mariadbDriver struct{}

func init() {
	RegisterDriver("mariadb", &mariadbDriver{})
}

func (d *mariadbDriver) Connect(instance config.DatabaseInstance) (*sql.DB, error) {
	dsn := instance.GetDSN()
	if dsn == "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			instance.Username, instance.Password, instance.Host, instance.Port, instance.Database)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func (d *mariadbDriver) GetStats(db *sql.DB, database string) (*stats.DatabaseStats, error) {
	result := &stats.DatabaseStats{
		Timestamp: time.Now(),
	}

	// Get status variables
	rows, err := db.Query("SHOW STATUS")
	if err != nil {
		return nil, fmt.Errorf("failed to get status variables: %w", err)
	}
	defer rows.Close()

	statusVars := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			continue
		}
		statusVars[name] = value
	}

	// Parse status variables
	if connections, ok := statusVars["Threads_connected"]; ok {
		if _, err := fmt.Sscanf(connections, "%d", &result.ActiveConnections); err != nil {
			result.ActiveConnections = 0
		}
	}

	if uptime, ok := statusVars["Uptime"]; ok {
		var uptimeSeconds int64
		if _, err := fmt.Sscanf(uptime, "%d", &uptimeSeconds); err == nil {
			result.Uptime = time.Duration(uptimeSeconds) * time.Second
		}
	}

	// Get process information
	processQuery := "SHOW PROCESSLIST"

	processRows, err := db.Query(processQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get process information: %w", err)
	}
	defer processRows.Close()

	for processRows.Next() {
		var process stats.ProcessInfo
		var timeStr string
		var state sql.NullString
		var info sql.NullString

		err := processRows.Scan(&process.ID, &process.User, &process.Host, &process.Database, &process.Command, &timeStr, &state, &info)
		if err != nil {
			continue
		}

		// Filter by database if specified
		if database != "" && process.Database != database {
			continue
		}

		if state.Valid {
			process.State = state.String
		}

		if info.Valid {
			process.Info = info.String
		}

		if timeStr != "" {
			if _, err := fmt.Sscanf(timeStr, "%d", &process.Time); err != nil {
				process.Time = 0
			}
		}

		result.Processes = append(result.Processes, process)
	}

	// Get table information
	tableQuery := `
		SELECT 
			table_name,
			table_rows,
			data_length,
			index_length
		FROM information_schema.tables 
	`
	if database != "" {
		tableQuery += " WHERE table_schema = ?"
		tableQuery += " ORDER BY (data_length + index_length) DESC LIMIT 10"
		tableRows, err := db.Query(tableQuery, database)
		if err != nil {
			return nil, fmt.Errorf("failed to get table information: %w", err)
		}
		defer tableRows.Close()

		for tableRows.Next() {
			var table stats.TableInfo
			var dataLength, indexLength sql.NullInt64

			err := tableRows.Scan(&table.Name, &table.Rows, &dataLength, &indexLength)
			if err != nil {
				continue
			}

			if dataLength.Valid {
				table.DataSize = dataLength.Int64
			}
			if indexLength.Valid {
				table.IndexSize = indexLength.Int64
			}

			result.Tables = append(result.Tables, table)
		}
	} else {
		// Show tables from all databases
		tableQuery += " ORDER BY (data_length + index_length) DESC LIMIT 20"
		tableRows, err := db.Query(tableQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to get table information: %w", err)
		}
		defer tableRows.Close()

		for tableRows.Next() {
			var table stats.TableInfo
			var schemaName string
			var dataLength, indexLength sql.NullInt64

			err := tableRows.Scan(&schemaName, &table.Name, &table.Rows, &dataLength, &indexLength)
			if err != nil {
				continue
			}

			table.Name = schemaName + "." + table.Name
			if dataLength.Valid {
				table.DataSize = dataLength.Int64
			}
			if indexLength.Valid {
				table.IndexSize = indexLength.Int64
			}

			result.Tables = append(result.Tables, table)
		}
	}

	return result, nil
}
