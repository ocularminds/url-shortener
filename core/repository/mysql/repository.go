// Package mysql provides the MySQL implementation of the repository contract.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/ocularminds/url-shortener/config"
	"github.com/ocularminds/url-shortener/core/models"
	"github.com/ocularminds/url-shortener/core/repository"
)

type Repository struct {
	db *sql.DB
}

// NewWithDB adapts an existing database pool. The caller retains responsibility
// for configuring its limits; Repository.Close closes the supplied pool.
func NewWithDB(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("database pool is required")
	}
	return &Repository{db: db}, nil
}

func New(ctx context.Context, cfg config.DatabaseConfig) (*Repository, error) {
	driverConfig := mysqldriver.NewConfig()
	driverConfig.User = cfg.Username
	driverConfig.Passwd = cfg.Password
	driverConfig.Net = "tcp"
	driverConfig.Addr = cfg.Address()
	driverConfig.DBName = cfg.Name
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	driverConfig.Timeout = cfg.ConnectTimeout
	driverConfig.ReadTimeout = cfg.ReadTimeout
	driverConfig.WriteTimeout = cfg.WriteTimeout
	driverConfig.TLSConfig = cfg.TLSMode

	connector, err := mysqldriver.NewConnector(driverConfig)
	if err != nil {
		return nil, fmt.Errorf("configure database: %w", err)
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)
	db.SetConnMaxLifetime(cfg.ConnectionLifetime)
	db.SetConnMaxIdleTime(cfg.ConnectionIdleTime)

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return NewWithDB(db)
}

func (store *Repository) Close() error {
	return store.db.Close()
}

func (store *Repository) FindBySlug(ctx context.Context, slug string) (models.ShortLink, error) {
	const query = `SELECT Shortened, Original, Expiry, Created, Hits
		FROM ShortLink WHERE Shortened = ? LIMIT 1`
	return store.find(ctx, query, slug)
}

func (store *Repository) FindByOriginal(ctx context.Context, original string) (models.ShortLink, error) {
	const query = `SELECT Shortened, Original, Expiry, Created, Hits
		FROM ShortLink WHERE Original = ? ORDER BY Created DESC LIMIT 1`
	return store.find(ctx, query, original)
}

func (store *Repository) find(ctx context.Context, query string, value string) (models.ShortLink, error) {
	var link models.ShortLink
	err := store.db.QueryRowContext(ctx, query, value).Scan(
		&link.Shortened,
		&link.Original,
		&link.Expiry,
		&link.Created,
		&link.Hits,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ShortLink{}, repository.ErrNotFound
	}
	if err != nil {
		return models.ShortLink{}, fmt.Errorf("find short link: %w", err)
	}
	return link, nil
}

func (store *Repository) Create(ctx context.Context, link models.ShortLink) error {
	const statement = `INSERT INTO ShortLink
		(Shortened, Original, Expiry, Created, Hits) VALUES (?, ?, ?, ?, ?)`
	_, err := store.db.ExecContext(
		ctx,
		statement,
		link.Shortened,
		link.Original,
		link.Expiry,
		link.Created,
		link.Hits,
	)
	var mysqlError *mysqldriver.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
		return repository.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("create short link: %w", err)
	}
	return nil
}

func (store *Repository) IncrementHits(ctx context.Context, slug string) error {
	const statement = `UPDATE ShortLink SET Hits = Hits + 1 WHERE Shortened = ?`
	result, err := store.db.ExecContext(ctx, statement, slug)
	if err != nil {
		return fmt.Errorf("increment hits: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if rows != 1 {
		return repository.ErrNotFound
	}
	return nil
}
