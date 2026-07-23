// Package migrations встраивает SQL-файлы миграций в бинарь, чтобы он мог
// применять их при запуске без доступа к исходному дереву во время выполнения.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
