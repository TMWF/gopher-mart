package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var MigrationsFS embed.FS

func RunMigrations(db *sql.DB, log *slog.Logger) error {
	logger := log.With(slog.String("op", "migrations.RunMigrations"))
	sourceDriver, err := iofs.New(MigrationsFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	dbDriver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("failed to create db driver: %w", err)
	}

	m, err := migrate.NewWithInstance(

		"iofs",
		sourceDriver,
		"pgx",
		dbDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("Database is up to date (no changes)")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	logger.Info("Migrations applied successfully")
	return nil
}
