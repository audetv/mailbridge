// Package version содержит информацию о версии сборки.
// Значения заполняются через ldflags при компиляции.
package version

var (
	// Version содержит номер версии (из git tag или "dev").
	Version = "dev"

	// Commit содержит хеш коммита (или "none").
	Commit = "none"

	// BuildTime содержит время сборки в UTC (или "unknown").
	BuildTime = "unknown"
)

// Info возвращает строку с полной информацией о версии.
func Info() string {
	return "mailbridge " + Version + " (commit: " + Commit + ", built: " + BuildTime + ")"
}

// Short возвращает краткую версию.
func Short() string {
	return Version
}
