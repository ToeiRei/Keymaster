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
	defer dbMock.Close()

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
	defer dbMock.Close()

	orig := sqlOpenFunc
	sqlOpenFunc = func(driverName, dsn string) (*sql.DB, error) { return dbMock, nil }
	defer func() { sqlOpenFunc = orig }()

	// Simulate PRAGMA optimize failing
	mock.ExpectExec("PRAGMA optimize").WillReturnError(errors.New("optimize fail"))

	if err := RunDBMaintenance("sqlite", "whatever"); err == nil {
		t.Fatalf("expected error when PRAGMA optimize fails")
	}
}
