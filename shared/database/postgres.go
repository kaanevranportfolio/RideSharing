package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rideshare-platform/shared/config"
	"github.com/rideshare-platform/shared/logger"
)

// PostgresDB represents a PostgreSQL database connection
type PostgresDB struct {
	DB     *sql.DB
	config *config.DatabaseConfig
	logger *logger.Logger
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.DatabaseConfig, log *logger.Logger) (*PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.WithFields(logger.Fields{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"database": cfg.Database,
	}).Info("Connected to PostgreSQL database")

	return &PostgresDB{
		DB:     db,
		config: cfg,
		logger: log,
	}, nil
}

// Close closes the database connection
func (p *PostgresDB) Close() error {
	if p.DB != nil {
		p.logger.Logger.Info("Closing PostgreSQL database connection")
		return p.DB.Close()
	}
	return nil
}

// Health checks the database health
func (p *PostgresDB) Health(ctx context.Context) error {
	return p.DB.PingContext(ctx)
}

// BeginTx starts a new transaction
func (p *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return p.DB.BeginTx(ctx, opts)
}

// ExecContext executes a query without returning any rows
func (p *PostgresDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := p.DB.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	p.logger.LogDatabaseQuery(ctx, query, duration, err)
	return result, err
}

// QueryContext executes a query that returns rows
func (p *PostgresDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := p.DB.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	p.logger.LogDatabaseQuery(ctx, query, duration, err)
	return rows, err
}

// QueryRowContext executes a query that is expected to return at most one row
func (p *PostgresDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := p.DB.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	p.logger.LogDatabaseQuery(ctx, query, duration, nil)
	return row
}

// PrepareContext creates a prepared statement for later queries or executions
func (p *PostgresDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return p.DB.PrepareContext(ctx, query)
}

// Stats returns database statistics
func (p *PostgresDB) Stats() sql.DBStats {
	return p.DB.Stats()
}

// LogStats logs database connection pool statistics
func (p *PostgresDB) LogStats(ctx context.Context) {
	stats := p.DB.Stats()
	p.logger.WithContext(ctx).WithFields(logger.Fields{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration,
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}).Info("PostgreSQL connection pool stats")
}

// Transaction represents a database transaction with logging
type Transaction struct {
	tx     *sql.Tx
	logger *logger.Logger
	ctx    context.Context
}

// NewTransaction creates a new transaction wrapper
func (p *PostgresDB) NewTransaction(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
	tx, err := p.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	p.logger.WithContext(ctx).Debug("Database transaction started")

	return &Transaction{
		tx:     tx,
		logger: p.logger,
		ctx:    ctx,
	}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	err := t.tx.Commit()
	if err != nil {
		t.logger.WithContext(t.ctx).WithError(err).Error("Failed to commit transaction")
	} else {
		t.logger.WithContext(t.ctx).Debug("Database transaction committed")
	}
	return err
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	err := t.tx.Rollback()
	if err != nil {
		t.logger.WithContext(t.ctx).WithError(err).Error("Failed to rollback transaction")
	} else {
		t.logger.WithContext(t.ctx).Debug("Database transaction rolled back")
	}
	return err
}

// ExecContext executes a query within the transaction
func (t *Transaction) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := t.tx.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	t.logger.LogDatabaseQuery(ctx, query, duration, err)
	return result, err
}

// QueryContext executes a query within the transaction
func (t *Transaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := t.tx.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	t.logger.LogDatabaseQuery(ctx, query, duration, err)
	return rows, err
}

// QueryRowContext executes a query within the transaction
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := t.tx.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	t.logger.LogDatabaseQuery(ctx, query, duration, nil)
	return row
}

// PrepareContext creates a prepared statement within the transaction
func (t *Transaction) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return t.tx.PrepareContext(ctx, query)
}

// WithTransaction executes a function within a database transaction
func (p *PostgresDB) WithTransaction(ctx context.Context, opts *sql.TxOptions, fn func(*Transaction) error) error {
	tx, err := p.NewTransaction(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			p.logger.WithContext(ctx).WithError(rbErr).Error("Failed to rollback transaction after error")
		}
		return err
	}

	return tx.Commit()
}

// Repository provides common database operations
type Repository struct {
	db     *PostgresDB
	logger *logger.Logger
}

// NewRepository creates a new repository
func NewRepository(db *PostgresDB, logger *logger.Logger) *Repository {
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Exists checks if a record exists
func (r *Repository) Exists(ctx context.Context, table, column string, value interface{}) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE %s = $1)", table, column)

	var exists bool
	err := r.db.QueryRowContext(ctx, query, value).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return exists, nil
}

// Count counts records matching a condition
func (r *Repository) Count(ctx context.Context, table, whereClause string, args ...interface{}) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count records: %w", err)
	}

	return count, nil
}

// GetLastInsertID gets the last inserted ID (PostgreSQL specific)
func (r *Repository) GetLastInsertID(ctx context.Context, table, idColumn string) (int64, error) {
	query := fmt.Sprintf("SELECT COALESCE(MAX(%s), 0) FROM %s", idColumn, table)

	var id int64
	err := r.db.QueryRowContext(ctx, query).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return id, nil
}
