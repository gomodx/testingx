package migrations

import (
	"embed"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/pkg/errors"
	"io/fs"
)

// FS exports the migrations file system for use in other areas of the program
// This allows the migrations to be embedded in the binary and used in testing
//
//go:embed drivers
var FS embed.FS

type MigrationParams struct {
	FS          fs.FS
	FSPath      string `json:"fs_path"`
	DatabaseURL string `json:"database_url"`
}

func NewMigrations(params MigrationParams) (*migrate.Migrate, error) {
	d, err := iofs.New(params.FS, params.FSPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize embedded migration file system")
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, params.DatabaseURL)
	if err != nil {
		return m, errors.Wrap(err, "failed to initialize migration package")
	}
	return m, nil
}
