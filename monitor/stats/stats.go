package stats

import "time"

// DatabaseStats represents database statistics
type DatabaseStats struct {
	Timestamp         time.Time
	ActiveConnections int64
	TotalConnections  int64
	QueriesPerSecond  float64
	SlowQueries       int64
	Uptime            time.Duration
	Threads           ThreadStats
	Processes         []ProcessInfo
	Tables            []TableInfo
}

// ThreadStats represents thread-related statistics
type ThreadStats struct {
	Running   int64
	Connected int64
	Sleeping  int64
	Locked    int64
}

// ProcessInfo represents information about a database process
type ProcessInfo struct {
	ID       int64
	User     string
	Host     string
	Database string
	Command  string
	Time     int64
	State    string
	Info     string
}

// TableInfo represents information about database tables
type TableInfo struct {
	Name      string
	Rows      int64
	DataSize  int64
	IndexSize int64
}
