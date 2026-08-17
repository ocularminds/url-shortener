package shortner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound = errors.New("short link not found")
	ErrConflict = errors.New("short link already exists")
)

type LinkRepository interface {
	FindBySlug(context.Context, string) (ShortLink, error)
	FindByOriginal(context.Context, string) (ShortLink, error)
	Create(context.Context, ShortLink) error
	IncrementHits(context.Context, string) error
}

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(ctx context.Context, cfg DatabaseConfig) (*MySQLRepository, error) {
	driverConfig := mysql.NewConfig()
	driverConfig.User = cfg.Username
	driverConfig.Passwd = cfg.Password
	driverConfig.Net = "tcp"
	driverConfig.Addr = cfg.address()
	driverConfig.DBName = cfg.Name
	driverConfig.ParseTime = true
	driverConfig.Loc = time.UTC
	driverConfig.Timeout = cfg.ConnectTimeout
	driverConfig.ReadTimeout = cfg.ReadTimeout
	driverConfig.WriteTimeout = cfg.WriteTimeout
	driverConfig.TLSConfig = cfg.TLSMode

	connector, err := mysql.NewConnector(driverConfig)
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
	return &MySQLRepository{db: db}, nil
}

func (repository *MySQLRepository) Close() error {
	return repository.db.Close()
}

func (repository *MySQLRepository) FindBySlug(ctx context.Context, slug string) (ShortLink, error) {
	const query = `SELECT Shortened, Original, Expiry, Created, Hits
		FROM ShortLink WHERE Shortened = ? LIMIT 1`
	return repository.find(ctx, query, slug)
}

func (repository *MySQLRepository) FindByOriginal(ctx context.Context, original string) (ShortLink, error) {
	const query = `SELECT Shortened, Original, Expiry, Created, Hits
		FROM ShortLink WHERE Original = ? ORDER BY Created DESC LIMIT 1`
	return repository.find(ctx, query, original)
}

func (repository *MySQLRepository) find(ctx context.Context, query string, value string) (ShortLink, error) {
	var link ShortLink
	err := repository.db.QueryRowContext(ctx, query, value).Scan(
		&link.Shortened,
		&link.Original,
		&link.Expiry,
		&link.Created,
		&link.Hits,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ShortLink{}, ErrNotFound
	}
	if err != nil {
		return ShortLink{}, fmt.Errorf("find short link: %w", err)
	}
	return link, nil
}

func (repository *MySQLRepository) Create(ctx context.Context, link ShortLink) error {
	const statement = `INSERT INTO ShortLink
		(Shortened, Original, Expiry, Created, Hits) VALUES (?, ?, ?, ?, ?)`
	_, err := repository.db.ExecContext(
		ctx,
		statement,
		link.Shortened,
		link.Original,
		link.Expiry,
		link.Created,
		link.Hits,
	)
	var mysqlError *mysql.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("create short link: %w", err)
	}
	return nil
}

func (repository *MySQLRepository) IncrementHits(ctx context.Context, slug string) error {
	const statement = `UPDATE ShortLink SET Hits = Hits + 1 WHERE Shortened = ?`
	result, err := repository.db.ExecContext(ctx, statement, slug)
	if err != nil {
		return fmt.Errorf("increment hits: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if rows != 1 {
		return ErrNotFound
	}
	return nil
}
