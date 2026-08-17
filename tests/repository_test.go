package tests

import (
	"context"
	"database/sql/driver"
	"errors"
	"net"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/ocularminds/url-shortener/config"
	"github.com/ocularminds/url-shortener/core/models"
	"github.com/ocularminds/url-shortener/core/repository"
	mysqlrepository "github.com/ocularminds/url-shortener/core/repository/mysql"
)

const (
	findBySlugQuery = `SELECT Shortened, Original, Expiry, Created, Hits
		FROM ShortLink WHERE Shortened = ? LIMIT 1`
	findByOriginalQuery = `SELECT Shortened, Original, Expiry, Created, Hits
		FROM ShortLink WHERE Original = ? ORDER BY Created DESC LIMIT 1`
	createStatement = `INSERT INTO ShortLink
		(Shortened, Original, Expiry, Created, Hits) VALUES (?, ?, ?, ?, ?)`
	incrementStatement = `UPDATE ShortLink SET Hits = Hits + 1 WHERE Shortened = ?`
)

type failingRowsResult struct{}

func (failingRowsResult) LastInsertId() (int64, error) { return 0, nil }
func (failingRowsResult) RowsAffected() (int64, error) { return 0, errors.New("rows unavailable") }

func newMockRepository(t *testing.T) (*mysqlrepository.Repository, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet SQL expectation: %v", err)
		}
		_ = database.Close()
	})
	store, err := mysqlrepository.NewWithDB(database)
	if err != nil {
		t.Fatal(err)
	}
	return store, mock
}

func TestRepositoryConstructorAndClose(t *testing.T) {
	if _, err := mysqlrepository.NewWithDB(nil); err == nil {
		t.Fatal("NewWithDB(nil) succeeded")
	}
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	store, err := mysqlrepository.NewWithDB(database)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectClose()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryNewReportsConfigurationAndConnectionFailures(t *testing.T) {
	base := config.Default().Database
	base.Name = "shortener"
	base.Username = "app"

	invalidTLS := base
	invalidTLS.TLSMode = "unregistered-test-profile"
	if _, err := mysqlrepository.New(context.Background(), invalidTLS); err == nil {
		t.Fatal("New() accepted an unregistered TLS profile")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, rawPort, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	disconnected := base
	disconnected.Port = port
	disconnected.ConnectTimeout = 100 * time.Millisecond
	if _, err := mysqlrepository.New(context.Background(), disconnected); err == nil {
		t.Fatal("New() connected to a closed local port")
	}
}

func TestRepositoryFindMethods(t *testing.T) {
	created := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	want := models.ShortLink{Shortened: "AbCd1234", Original: "https://example.com", Expiry: 30, Created: created, Hits: 7}

	tests := []struct {
		name  string
		query string
		value string
		find  func(*mysqlrepository.Repository) (models.ShortLink, error)
	}{
		{name: "by slug", query: findBySlugQuery, value: want.Shortened, find: func(store *mysqlrepository.Repository) (models.ShortLink, error) {
			return store.FindBySlug(context.Background(), want.Shortened)
		}},
		{name: "by original", query: findByOriginalQuery, value: want.Original, find: func(store *mysqlrepository.Repository) (models.ShortLink, error) {
			return store.FindByOriginal(context.Background(), want.Original)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, mock := newMockRepository(t)
			rows := sqlmock.NewRows([]string{"Shortened", "Original", "Expiry", "Created", "Hits"}).
				AddRow(want.Shortened, want.Original, want.Expiry, want.Created, want.Hits)
			mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.value).WillReturnRows(rows)

			got, err := test.find(store)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("find result = %+v, want %+v", got, want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRepositoryFindErrors(t *testing.T) {
	tests := []struct {
		name       string
		queryError error
		want       error
	}{
		{name: "missing", want: repository.ErrNotFound},
		{name: "database failure", queryError: errors.New("database unavailable")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, mock := newMockRepository(t)
			if test.queryError == nil {
				mock.ExpectQuery(regexp.QuoteMeta(findBySlugQuery)).WithArgs("Missing1").WillReturnRows(
					sqlmock.NewRows([]string{"Shortened", "Original", "Expiry", "Created", "Hits"}),
				)
			} else {
				mock.ExpectQuery(regexp.QuoteMeta(findBySlugQuery)).WithArgs("Missing1").WillReturnError(test.queryError)
			}

			_, err := store.FindBySlug(context.Background(), "Missing1")
			if test.want != nil && !errors.Is(err, test.want) {
				t.Fatalf("FindBySlug() error = %v, want %v", err, test.want)
			}
			if test.want == nil && err == nil {
				t.Fatal("FindBySlug() succeeded after a database failure")
			}
		})
	}
}

func TestRepositoryCreate(t *testing.T) {
	link := models.ShortLink{
		Shortened: "AbCd1234",
		Original:  "https://example.com",
		Expiry:    30,
		Created:   time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC),
		Hits:      2,
	}
	tests := []struct {
		name       string
		result     driver.Result
		execError  error
		wantError  error
		wantAnyErr bool
	}{
		{name: "success", result: sqlmock.NewResult(1, 1)},
		{name: "duplicate", execError: &mysqldriver.MySQLError{Number: 1062, Message: "duplicate"}, wantError: repository.ErrConflict},
		{name: "database failure", execError: errors.New("database unavailable"), wantAnyErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, mock := newMockRepository(t)
			expectation := mock.ExpectExec(regexp.QuoteMeta(createStatement)).
				WithArgs(link.Shortened, link.Original, link.Expiry, link.Created, link.Hits)
			if test.execError != nil {
				expectation.WillReturnError(test.execError)
			} else {
				expectation.WillReturnResult(test.result)
			}

			err := store.Create(context.Background(), link)
			if test.wantError != nil && !errors.Is(err, test.wantError) {
				t.Fatalf("Create() error = %v, want %v", err, test.wantError)
			}
			if test.wantAnyErr && err == nil {
				t.Fatal("Create() succeeded after a database failure")
			}
			if test.wantError == nil && !test.wantAnyErr && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRepositoryIncrementHits(t *testing.T) {
	tests := []struct {
		name       string
		result     driver.Result
		execError  error
		wantError  error
		wantAnyErr bool
	}{
		{name: "success", result: sqlmock.NewResult(0, 1)},
		{name: "missing", result: sqlmock.NewResult(0, 0), wantError: repository.ErrNotFound},
		{name: "unexpected row count", result: sqlmock.NewResult(0, 2), wantError: repository.ErrNotFound},
		{name: "result failure", result: failingRowsResult{}, wantAnyErr: true},
		{name: "database failure", execError: errors.New("database unavailable"), wantAnyErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, mock := newMockRepository(t)
			expectation := mock.ExpectExec(regexp.QuoteMeta(incrementStatement)).WithArgs("AbCd1234")
			if test.execError != nil {
				expectation.WillReturnError(test.execError)
			} else {
				expectation.WillReturnResult(test.result)
			}

			err := store.IncrementHits(context.Background(), "AbCd1234")
			if test.wantError != nil && !errors.Is(err, test.wantError) {
				t.Fatalf("IncrementHits() error = %v, want %v", err, test.wantError)
			}
			if test.wantAnyErr && err == nil {
				t.Fatal("IncrementHits() succeeded after a database failure")
			}
			if test.wantError == nil && !test.wantAnyErr && err != nil {
				t.Fatal(err)
			}
		})
	}
}
