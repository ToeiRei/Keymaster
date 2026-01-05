// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRunDBMaintenance_Sqlite_WithMock_Success(t *testing.T) {
	// create sqlmock DB
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = dbMock.Close() }()

	// override sqlOpenFunc to return our mock regardless of args
	orig := sqlOpenFunc
	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) { return dbMock, nil }
	defer func() { sqlOpenFunc = orig }()

	// Expect PRAGMA optimize; VACUUM; integrity_check
	mock.ExpectExec("PRAGMA optimize").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("VACUUM").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("PRAGMA wal_checkpoint\\(").WillReturnResult(sqlmock.NewResult(0, 0))
	rows := sqlmock.NewRows([]string{"integrity_check"}).AddRow("ok")
	mock.ExpectQuery("PRAGMA integrity_check").WillReturnRows(rows)

	if err := RunDBMaintenance("sqlite", "whatever"); err != nil {
		t.Fatalf("expected RunDBMaintenance success, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunDBMaintenance_Sqlite_WithMock_Failure(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = dbMock.Close() }()

	orig := sqlOpenFunc
	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) { return dbMock, nil }
	defer func() { sqlOpenFunc = orig }()

	// Simulate PRAGMA optimize failing
	mock.ExpectExec("PRAGMA optimize").WillReturnError(errors.New("optimize fail"))

	if err := RunDBMaintenance("sqlite", "whatever"); err == nil {
		t.Fatalf("expected error when PRAGMA optimize fails")
	}
}

func TestRunDBMaintenance_Postgres_WithMock_Success(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = dbMock.Close() }()

	orig := sqlOpenFunc
	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) { return dbMock, nil }
	defer func() { sqlOpenFunc = orig }()

	mock.ExpectExec("VACUUM ANALYZE").WillReturnResult(sqlmock.NewResult(0, 0))

	if err := RunDBMaintenance("postgres", "dsn"); err != nil {
		t.Fatalf("expected postgres maintenance to succeed, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunDBMaintenance_MySQL_WithMock_Success(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = dbMock.Close() }()

	orig := sqlOpenFunc
	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) { return dbMock, nil }
	defer func() { sqlOpenFunc = orig }()

	// Return one table name from SHOW TABLES
	rows := sqlmock.NewRows([]string{"Tables_in_db"}).AddRow("users")
	mock.ExpectQuery("SHOW TABLES").WillReturnRows(rows)
	mock.ExpectExec("OPTIMIZE TABLE users").WillReturnResult(sqlmock.NewResult(0, 0))

	if err := RunDBMaintenance("mysql", "dsn"); err != nil {
		t.Fatalf("expected mysql maintenance to succeed, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunDBMaintenance_Postgres_WithMock_Failure(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = dbMock.Close() }()

	orig := sqlOpenFunc
	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) { return dbMock, nil }
	defer func() { sqlOpenFunc = orig }()

	mock.ExpectExec("VACUUM ANALYZE").WillReturnError(errors.New("vacuum fail"))

	if err := RunDBMaintenance("postgres", "dsn"); err == nil {
		t.Fatalf("expected error when VACUUM ANALYZE fails")
	}
}

func TestRunDBMaintenance_MySQL_WithMock_Failure(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = dbMock.Close() }()

	orig := sqlOpenFunc
	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) { return dbMock, nil }
	defer func() { sqlOpenFunc = orig }()

	rows := sqlmock.NewRows([]string{"Tables_in_db"}).AddRow("users")
	mock.ExpectQuery("SHOW TABLES").WillReturnRows(rows)
	mock.ExpectExec("OPTIMIZE TABLE users").WillReturnError(errors.New("optimize fail"))

	if err := RunDBMaintenance("mysql", "dsn"); err == nil {
		t.Fatalf("expected error when OPTIMIZE TABLE fails")
	}
}
