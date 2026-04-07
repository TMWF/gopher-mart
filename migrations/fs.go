package migrations

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var MigrationsFS embed.FS

func RunMigrations(databaseURL string, logger *slog.Logger) error {
	log := logger.With(slog.String("op", "migrations.RunMigrations"))
	d, err := iofs.New(MigrationsFS, ".")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	// migrateURL := "pgx5://" + databaseURL[len("postgres://"):]
	migrateURL := getDsn(databaseURL)

	m, err := migrate.NewWithSourceInstance("iofs", d, migrateURL)
	if err != nil {
		return fmt.Errorf("failed to initialize migrate: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrate up: %w", err)
	}

	log.Info("Migrations applied successfully!")
	return nil
	// sourceDriver, err := iofs.New(MigrationsFS, ".")
	// if err != nil {
	// 	return fmt.Errorf("failed to create iofs driver: %w", err)
	// }

	// dbDriver, err := pgx.WithInstance(db, &pgx.Config{})
	// if err != nil {
	// 	return fmt.Errorf("failed to create db driver: %w", err)
	// }

	// m, err := migrate.NewWithInstance(
	// 	"iofs",
	// 	sourceDriver,
	// 	"pgx",
	// 	dbDriver,
	// )
	// if err != nil {
	// 	return fmt.Errorf("failed to create migrate instance %w", err)
	// }

	// if err := m.Up(); err != nil {
	// 	if errors.Is(err, migrate.ErrNoChange) {
	// 		logger.Info("Database is up to date (no changes)")
	// 		return nil
	// 	}
	// 	return fmt.Errorf("failed to apply migrations: %w", err)
	// }

	// logger.Info("Migrations applied successfully")
	// return nil
}

func getDsn(dsn string) string {
	if strings.Contains(dsn, "://") {
		dsn = strings.Replace(dsn, "postgres://", "pgx5://", 1)
		dsn = strings.Replace(dsn, "postgresql://", "pgx5://", 1)
		return dsn
	}

	return "pgx5://" + dsn
}
