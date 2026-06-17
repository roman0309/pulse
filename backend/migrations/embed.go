// Package migrations embeds the SQL migration files so the backend can apply
// them on startup — no mounted files or external migration tool required.
package migrations

import "embed"

//go:embed postgres/*.sql clickhouse/*.sql
var FS embed.FS
