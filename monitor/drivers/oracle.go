package drivers

import (
	"database/sql"
	"fmt"
	"time"

	"dbtop/config"
	"dbtop/monitor/stats"

	_ "github.com/godror/godror"
)

type oracleDriver struct{}

func init() {
	RegisterDriver("oracle", &oracleDriver{})
}

func (d *oracleDriver) Connect(instance config.DatabaseInstance) (*sql.DB, error) {
	dsn := instance.GetDSN()
	if dsn == "" {
		dsn = fmt.Sprintf("%s/%s@%s:%d/%s",
			instance.Username, instance.Password, instance.Host, instance.Port, instance.Database)
	}

	db, err := sql.Open("godror", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func (d *oracleDriver) GetStats(db *sql.DB, database string) (*stats.DatabaseStats, error) {
	stats := &stats.DatabaseStats{
		Timestamp: time.Now(),
	}

	// Get active sessions
	var activeConnections int64
	activeQuery := `
		SELECT COUNT(*) 
		FROM v$session 
		WHERE status = 'ACTIVE' AND username IS NOT NULL
	`
	if database != "" {
		activeQuery += " AND schemaname = :1"
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

	// Get total sessions
	var totalConnections int64
	totalQuery := `
		SELECT COUNT(*) 
		FROM v$session 
		WHERE username IS NOT NULL
	`
	if database != "" {
		totalQuery += " AND schemaname = :1"
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
	err := db.QueryRow(`
		SELECT EXTRACT(DAY FROM (SYSDATE - STARTUP_TIME)) * 86400 +
		       EXTRACT(HOUR FROM (SYSDATE - STARTUP_TIME)) * 3600 +
		       EXTRACT(MINUTE FROM (SYSDATE - STARTUP_TIME)) * 60 +
		       EXTRACT(SECOND FROM (SYSDATE - STARTUP_TIME))
		FROM v$instance
	`).Scan(&uptimeSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime: %w", err)
	}
	stats.Uptime = time.Duration(uptimeSeconds) * time.Second

	// Get session information
	sessionQuery := `
		SELECT 
			s.sid,
			s.username,
			s.machine,
			s.schemaname,
			s.status,
			s.logon_time,
			q.sql_text
		FROM v$session s
		LEFT JOIN v$sql q ON s.sql_id = q.sql_id
		WHERE s.username IS NOT NULL
	`
	if database != "" {
		sessionQuery += " AND s.schemaname = :1"
	}
	sessionQuery += " ORDER BY s.logon_time DESC FETCH FIRST 50 ROWS ONLY"

	var rows *sql.Rows
	var err2 error
	if database != "" {
		rows, err2 = db.Query(sessionQuery, database)
	} else {
		rows, err2 = db.Query(sessionQuery)
	}
	if err2 != nil {
		return nil, fmt.Errorf("failed to get session information: %w", err2)
	}
	defer rows.Close()

	for rows.Next() {
		var process stats.ProcessInfo
		var logonTime sql.NullTime
		var sqlText sql.NullString

		err := rows.Scan(&process.ID, &process.User, &process.Host, &process.Database, &process.State, &logonTime, &sqlText)
		if err != nil {
			continue
		}

		if logonTime.Valid {
			process.Time = int64(time.Since(logonTime.Time).Seconds())
		}

		if sqlText.Valid {
			process.Info = sqlText.String
		}

		stats.Processes = append(stats.Processes, process)
	}

	// Get table information
	tableQuery := `
		SELECT 
			t.table_name,
			t.num_rows,
			s.bytes as data_size,
			i.index_size
		FROM user_tables t
		LEFT JOIN (
			SELECT segment_name, SUM(bytes) as bytes
			FROM user_segments
			WHERE segment_type = 'TABLE'
			GROUP BY segment_name
		) s ON t.table_name = s.segment_name
		LEFT JOIN (
			SELECT table_name, SUM(bytes) as index_size
			FROM user_segments
			WHERE segment_type = 'INDEX'
			GROUP BY table_name
		) i ON t.table_name = i.table_name
		ORDER BY (s.bytes + i.index_size) DESC
		FETCH FIRST 10 ROWS ONLY
	`
	if database != "" {
		// For Oracle, we need to connect to the specific schema
		// This is a simplified approach - in practice, you might need to switch schemas
		tableQuery = `
			SELECT 
				t.table_name,
				t.num_rows,
				s.bytes as data_size,
				i.index_size
			FROM all_tables t
			LEFT JOIN (
				SELECT segment_name, SUM(bytes) as bytes
				FROM all_segments
				WHERE segment_type = 'TABLE' AND owner = :1
				GROUP BY segment_name
			) s ON t.table_name = s.segment_name
			LEFT JOIN (
				SELECT table_name, SUM(bytes) as index_size
				FROM all_segments
				WHERE segment_type = 'INDEX' AND owner = :1
				GROUP BY table_name
			) i ON t.table_name = i.table_name
			WHERE t.owner = :1
			ORDER BY (s.bytes + i.index_size) DESC
			FETCH FIRST 10 ROWS ONLY
		`
		tableRows, err := db.Query(tableQuery, database)
		if err != nil {
			return nil, fmt.Errorf("failed to get table information: %w", err)
		}
		defer tableRows.Close()

		for tableRows.Next() {
			var table stats.TableInfo
			var dataSize, indexSize sql.NullInt64

			err := tableRows.Scan(&table.Name, &table.Rows, &dataSize, &indexSize)
			if err != nil {
				continue
			}

			if dataSize.Valid {
				table.DataSize = dataSize.Int64
			}
			if indexSize.Valid {
				table.IndexSize = indexSize.Int64
			}

			stats.Tables = append(stats.Tables, table)
		}
	} else {
		// Show tables from all schemas (limited to user's accessible schemas)
		tableQuery = `
			SELECT 
				t.owner || '.' || t.table_name,
				t.num_rows,
				s.bytes as data_size,
				i.index_size
			FROM all_tables t
			LEFT JOIN (
				SELECT owner, segment_name, SUM(bytes) as bytes
				FROM all_segments
				WHERE segment_type = 'TABLE'
				GROUP BY owner, segment_name
			) s ON t.owner = s.owner AND t.table_name = s.segment_name
			LEFT JOIN (
				SELECT owner, table_name, SUM(bytes) as index_size
				FROM all_segments
				WHERE segment_type = 'INDEX'
				GROUP BY owner, table_name
			) i ON t.owner = i.owner AND t.table_name = i.table_name
			ORDER BY (s.bytes + i.index_size) DESC
			FETCH FIRST 20 ROWS ONLY
		`
		tableRows, err := db.Query(tableQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to get table information: %w", err)
		}
		defer tableRows.Close()

		for tableRows.Next() {
			var table stats.TableInfo
			var dataSize, indexSize sql.NullInt64

			err := tableRows.Scan(&table.Name, &table.Rows, &dataSize, &indexSize)
			if err != nil {
				continue
			}

			if dataSize.Valid {
				table.DataSize = dataSize.Int64
			}
			if indexSize.Valid {
				table.IndexSize = indexSize.Int64
			}

			stats.Tables = append(stats.Tables, table)
		}
	}

	return stats, nil
}
