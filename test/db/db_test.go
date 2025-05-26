package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmticonap/real-logs/domain"
	"github.com/jmticonap/real-logs/infrastructure/db"
	"github.com/stretchr/testify/assert"
)

func TestOpenDb_DefaultPath(t *testing.T) {
	// Arrange
	params := domain.StrObject{}
	defaultDbPath := "./log.db"

	// Cleanup any existing default database file
	os.Remove(defaultDbPath)

	// Act
	database := db.OpenDb(params)

	// Assert
	assert.NotNil(t, database, "Database should not be nil")
	assert.NoError(t, database.Ping(), "Should be able to ping the database")

	// Check if the file was created
	_, err := os.Stat(defaultDbPath)
	assert.NoError(t, err, "Default database file should be created")

	// Cleanup
	database.Close()
	os.Remove(defaultDbPath)
}

func TestOpenDb_CustomPath(t *testing.T) {
	// Arrange
	testDir := "./test_db_dir"
	customDbPath := filepath.Join(testDir, "log.db")
	params := domain.StrObject{"dir": testDir}

	// Create test directory if it doesn't exist
	os.MkdirAll(testDir, 0777)
	defer os.RemoveAll(testDir) // Cleanup after test

	// Act
	database := db.OpenDb(params)

	// Assert
	assert.NotNil(t, database, "Database should not be nil")
	assert.NoError(t, database.Ping(), "Should be able to ping the database")

	// Check if the file was created in the custom path
	_, err := os.Stat(customDbPath)
	assert.NoError(t, err, "Custom database file should be created")

	// Cleanup
	database.Close()
}

func TestOpenDb_Singleton(t *testing.T) {
	// Arrange
	params := domain.StrObject{}
	defaultDbPath := "./log.db"
	os.Remove(defaultDbPath) // Ensure no existing file

	// Act
	db1 := db.OpenDb(params)
	db2 := db.OpenDb(params)

	// Assert
	assert.NotNil(t, db1, "First database instance should not be nil")
	assert.NotNil(t, db2, "Second database instance should not be nil")
	assert.Equal(t, db1, db2, "Should return the same database instance (singleton)")

	// Cleanup
	db1.Close()
	os.Remove(defaultDbPath)
}

func TestOpenDb_TablesCreated(t *testing.T) {
	// Arrange
	params := domain.StrObject{"dir": "./tables"}
	defaultDbPath := "./tables/log.db"
	os.Remove(defaultDbPath)

	// Act
	database := db.OpenDb(params)
	defer database.Close()
	defer os.Remove(defaultDbPath)

	// Assert
	assert.NotNil(t, database, "Database should not be nil")

	// Verify if performance_logs table exists
	_, err := database.ExecContext(context.Background(), "SELECT * FROM performance_logs LIMIT 1")
	assert.NoError(t, err, "performance_logs table should exist")

	// Verify if general_logs table exists
	_, err = database.ExecContext(context.Background(), "SELECT * FROM general_logs LIMIT 1")
	assert.NoError(t, err, "general_logs table should exist")
}

func TestOpenDb_ExistingDb(t *testing.T) {
	// Arrange
	defaultDbPath := "./exist/log.db"
	params := domain.StrObject{"dir": "exist"}

	// create a db
	db.OpenDb(params)
	defer os.Remove(defaultDbPath)

	// Act
	database := db.OpenDb(params)
	defer database.Close()

	// Assert
	assert.NotNil(t, database, "Database should not be nil")
	assert.NoError(t, database.Ping(), "Should be able to ping the existing database")

	// Verify tables are (re)created - attempt to query
	_, err := database.ExecContext(context.Background(), "SELECT * FROM performance_logs LIMIT 1")
	assert.NoError(t, err, "performance_logs table should exist in existing db")
}
