package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"slices"

	_ "github.com/lib/pq"
	"github.com/nikuIin/base_go_auth/src/core"
	"github.com/pressly/goose"
)


func ConnectToDatabase(databaseConfig core.DatabaseConfig, logger *slog.Logger) (*sql.DB, error) {

	if isDatabaseDriverAllowed(databaseConfig.DBDriver) == false {
		logger.Error("Unsupported database driver.", "database_driver", databaseConfig.DBDriver)
		return nil, fmt.Errorf("Unsupported database driver: %v", databaseConfig.DBDriver)
	}


	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		databaseConfig.Host,
		databaseConfig.Port,
	 	databaseConfig.Username,
		databaseConfig.Password,
	 	databaseConfig.DBName,
	)

	db, err := sql.Open(databaseConfig.DBDriver, connStr)

	if err != nil {
		logger.Error("Failed connect to DB.")
		return nil, fmt.Errorf("Failed to open database. Check your configuration.",)
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		logger.Error("Failed check connection to DB", "err", err)
		return nil, fmt.Errorf("Failed to ping database: %v", err)
	}

	// run migrations
	if err := goose.Up(db, "./migrations"); err != nil {
    		logger.Error("goose up failed: %v", err)
    	}

	logger.Debug("Successfully connected to Database.")
	return db, nil
}

func isDatabaseDriverAllowed(driver string) bool {
	var allowedDrivers = []string{
		"postgres",
	}

	return slices.Contains(allowedDrivers, driver)
}
