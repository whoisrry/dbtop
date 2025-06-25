package drivers

import (
	"database/sql"
	"fmt"
	"time"

	"dbtop/config"
	"dbtop/monitor/stats"

	_ "github.com/lib/pq"
)

type postgresDriver struct{}

func init() {
	RegisterDriver("postgres", &postgresDriver{})
	RegisterDriver("postgresql", &postgresDriver{})
}

func (d *postgresDriver) Connect(instance config.DatabaseInstance) (*sql.DB, error) {
	dsn := instance.GetDSN()
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			instance.Host, instance.Port, instance.Username, instance.Password, instance.Database, instance.SSLMode)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func (d *postgresDriver) GetStats(db *sql.DB, database string) (*stats.DatabaseStats, error) {
	stats := &stats.DatabaseStats{
		Timestamp: time.Now(),
	}

	// Get active connections
	var activeConnections int64
	activeQuery := "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"
	if database != "" {
		activeQuery += " AND datname = $1"
		err := db.QueryRow(activeQuery, database).Scan(&activeConnections)
		if err != nil {
			return nil, fmt.Errorf("failed to get active connections: %w", err)
		}
	} else {
		err := db.QueryRow(activeQuery).Scan(&activeConnections)
		if err != nil {
			return nil, fmt.Errorf("failed to get active connections: %w", err)
		}
	}
	stats.ActiveConnections = activeConnections

	// Get total connections
	var totalConnections int64
	totalQuery := "SELECT count(*) FROM pg_stat_activity"
	if database != "" {
		totalQuery += " WHERE datname = $1"
		err := db.QueryRow(totalQuery, database).Scan(&totalConnections)
		if err != nil {
			return nil, fmt.Errorf("failed to get total connections: %w", err)
		}
	} else {
		err := db.QueryRow(totalQuery).Scan(&totalConnections)
		if err != nil {
			return nil, fmt.Errorf("failed to get total connections: %w", err)
		}
	}
	stats.TotalConnections = totalConnections

	// Get uptime
	var uptimeSeconds int64
	err := db.QueryRow("SELECT EXTRACT(EPOCH FROM (now() - pg_postmaster_start_time()))").Scan(&uptimeSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime: %w", err)
	}
	stats.Uptime = time.Duration(uptimeSeconds) * time.Second

	// Get process information
	processQuery := `
		SELECT 
			pid,
			usename,
			client_addr,
			datname,
			state,
			query_start,
			query
		FROM pg_stat_activity 
		WHERE state IS NOT NULL
	`
	if database != "" {
		processQuery += " AND datname = $1"
	}
	processQuery += " ORDER BY query_start DESC LIMIT 50"

	var rows *sql.Rows
	var err2 error
	if database != "" {
		rows, err2 = db.Query(processQuery, database)
	} else {
		rows, err2 = db.Query(processQuery)
	}
	if err2 != nil {
		return nil, fmt.Errorf("failed to get process information: %w", err2)
	}
	defer rows.Close()

	for rows.Next() {
		var process stats.ProcessInfo
		var queryStart sql.NullTime
		var query sql.NullString
		var clientAddr sql.NullString

		err := rows.Scan(&process.ID, &process.User, &clientAddr, &process.Database, &process.State, &queryStart, &query)
		if err != nil {
			continue
		}

		if clientAddr.Valid {
			process.Host = clientAddr.String
		} else {
			process.Host = "localhost"
		}

		if query.Valid {
			process.Info = query.String
		}

		if queryStart.Valid {
			process.Time = int64(time.Since(queryStart.Time).Seconds())
		}

		stats.Processes = append(stats.Processes, process)
	}

	// Get table information
	tableQuery := `
		SELECT 
			schemaname || '.' || tablename as table_name,
			n_tup_ins + n_tup_upd + n_tup_del as total_rows,
			pg_total_relation_size(schemaname || '.' || tablename) as total_size
		FROM pg_stat_user_tables 
	`
	if database != "" {
		tableQuery += " WHERE schemaname = 'public'"
	}
	tableQuery += " ORDER BY total_size DESC LIMIT 10"

	var tableRows *sql.Rows
	var err3 error
	if database != "" {
		tableRows, err3 = db.Query(tableQuery)
	} else {
		tableRows, err3 = db.Query(tableQuery)
	}
	if err3 != nil {
		return nil, fmt.Errorf("failed to get table information: %w", err3)
	}
	defer tableRows.Close()

	for tableRows.Next() {
		var table stats.TableInfo
		var totalSize int64

		err := tableRows.Scan(&table.Name, &table.Rows, &totalSize)
		if err != nil {
			continue
		}

		table.DataSize = totalSize
		stats.Tables = append(stats.Tables, table)
	}

	return stats, nil
}
