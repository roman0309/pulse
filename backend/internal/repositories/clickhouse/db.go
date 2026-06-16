package clickhouse

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Connect opens a ClickHouse connection from a clickhouse:// DSN, retrying to
// tolerate the server still booting under docker-compose.
func Connect(ctx context.Context, dsn string) (driver.Conn, error) {
	opts, err := clickhouse.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}

	var conn driver.Conn
	for i := 0; i < 15; i++ {
		conn, err = clickhouse.Open(opts)
		if err == nil {
			if err = conn.Ping(ctx); err == nil {
				return conn, nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return nil, err
}
