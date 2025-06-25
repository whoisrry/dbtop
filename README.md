# dbtop - Database Monitoring Tool

A modular database monitoring tool written in Go that can monitor different database engines like PostgreSQL, MySQL, MariaDB, and Oracle.

> **Note**: This program was created with the assistance of AI tools. The author is not a frequent coder, so AI was used to help design the architecture, implement the features, and ensure best practices.

## Features

- **Multi-database support**: Monitor PostgreSQL, MySQL, MariaDB, and Oracle databases
- **Modular architecture**: Each database type has its own driver implementation
- **Real-time monitoring**: Live updates of database statistics and processes
- **Terminal UI**: Beautiful terminal-based interface using termui
- **Configuration-based**: Easy configuration through YAML files
- **All-database monitoring**: Like mytop, can monitor all databases when no specific database is set
- **Dynamic height adjustment**: Automatically fits to terminal height
- **Sorting and filtering**: Sort processes by ID, user, host, database, time, or state
- **Refresh rate control**: Adjustable refresh intervals via config or keyboard shortcuts

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd dbtop
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o dbtop
```

4. Install globally (optional):
```bash
sudo cp dbtop /usr/local/bin/
```

## Configuration

Create a configuration file at `~/.dbtop` with your database instances:

```yaml
instances:
  # Monitor specific database
  db1:
    type: postgres
    host: localhost
    port: 5432
    username: postgres
    password: your_password
    database: your_database  # Optional - if not set, monitors all databases
    ssl_mode: disable
    refresh_interval: 2s     # Optional - default is 2s

  # Monitor all databases (like mytop)
  db2:
    type: mysql
    host: localhost
    port: 3306
    username: root
    password: your_password
    # database: not set - will monitor all databases
    refresh_interval: 1s

  db3:
    type: mariadb
    host: localhost
    port: 3306
    username: root
    password: your_password
    database: your_database

  db4:
    type: oracle
    host: localhost
    port: 1521
    username: system
    password: your_password
    database: XE
```

## Usage

### Monitor a specific database instance:
```bash
dbtop db1
```

### Monitor the default instance (if only one is configured):
```bash
dbtop
```

### List available instances:
```bash
dbtop
# This will show available instances if multiple are configured
```

## Keyboard Controls

- **q** or **Ctrl+C**: Quit the application
- **s**: Cycle through sort fields (ID, User, Host, Database, Time, State)
- **r**: Reverse sort order
- **+**: Increase refresh rate (decrease interval)
- **-**: Decrease refresh rate (increase interval)
- **h**: Show help (displays current controls)

## Supported Database Types

### PostgreSQL
- Active connections
- Total connections
- Uptime
- Process information
- Table statistics
- Optional database filtering

### MySQL
- Thread status
- Connection information
- Process list
- Table information
- Optional database filtering

### MariaDB
- Thread status
- Connection information
- Process list
- Table information
- Optional database filtering

### Oracle
- Session information
- Connection statistics
- SQL execution details
- Table and index sizes
- Optional schema filtering

## UI Features

The terminal UI displays:
- **Connection Info**: Instance name, database type, uptime, active connections, and refresh interval
- **Database Statistics**: Various metrics like total connections, queries per second, etc.
- **Active Processes**: Real-time list of database processes and their states
- **Dynamic Height**: Automatically adjusts to fit your terminal height
- **Sorting**: Sort processes by different fields
- **Refresh Control**: Adjustable refresh intervals

## Configuration Options

### Database Instance Configuration

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Database type: `postgres`, `postgresql`, `mysql`, `mariadb`, `oracle` |
| `host` | string | Yes | Database host |
| `port` | int | Yes | Database port |
| `username` | string | Yes | Database username |
| `password` | string | Yes | Database password |
| `database` | string | No | Database name (if not set, monitors all databases) |
| `ssl_mode` | string | No | SSL mode (PostgreSQL only) |
| `refresh_interval` | duration | No | Refresh interval (default: 2s) |
| `options` | map | No | Additional database-specific options |

## Dependencies

- `github.com/lib/pq` - PostgreSQL driver
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/godror/godror` - Oracle driver
- `github.com/gizak/termui/v3` - Terminal UI
- `gopkg.in/yaml.v3` - YAML configuration parsing

## Building from Source

```bash
# Install dependencies
go mod download

# Build for current platform
go build -o dbtop

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o dbtop-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o dbtop-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o dbtop-windows-amd64.exe
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the GNU General Public License v3.0 (GPLv3). See the LICENSE file for details. 