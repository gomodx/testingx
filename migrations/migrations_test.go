package migrations

import (
	"github.com/sourcec0de/testingx/database"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMigrations(t *testing.T) {
	pgt, err := database.NewPostgresTestInstance(nil, nil)
	defer pgt.Cleanup()
	require.NoError(t, err)

	migrations, err := NewMigrations(MigrationParams{
		FS:          FS,
		FSPath:      "drivers/postgres",
		DatabaseURL: pgt.DatabaseURL,
	})

	require.NoError(t, err)

	err = migrations.Up()
	require.NoError(t, err)

	err = migrations.Down()
	require.NoError(t, err)
}
