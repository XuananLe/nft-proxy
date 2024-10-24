package services

import (
    "context"
    "errors"
    "fmt"
    "log"
    "os"
    "sync"
    "time"

    nft_proxy "github.com/alphabatem/nft-proxy"
    "github.com/babilu-online/common/context"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

// Custom error types
var (
    ErrDatabaseNotConfigured = errors.New("database not configured")
    ErrConnectionFailed     = errors.New("failed to connect to database")
    ErrTransactionFailed    = errors.New("transaction failed")
)

// SqliteError provides context-rich error information
// ISSUE: Original implementation mixed HTTP status codes with database errors
type SqliteError struct {
    Code    int
    Message string
    Err     error
}

func (e *SqliteError) Error() string {
    return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

// SqliteConfig holds all database configuration
// ISSUE: Original implementation had unused credentials and lacked configuration structure
type SqliteConfig struct {
    Database     string
    MaxIdleConns int
    MaxOpenConns int
    MaxLifetime  time.Duration
    QueryTimeout time.Duration
}

// DefaultConfig provides sensible defaults
func DefaultConfig() SqliteConfig {
    return SqliteConfig{
        MaxIdleConns: 10,
        MaxOpenConns: 100,
        MaxLifetime:  time.Hour,
        QueryTimeout: time.Second * 30,
    }
}

// SqliteService provides SQLite database operations
// ISSUE: Original lacked proper connection management and configuration
type SqliteService struct {
    context.Context
    mu     sync.RWMutex
    db     *gorm.DB
    config SqliteConfig
}

const SQLITE_SVC = "sqlite_svc"

// Id returns Service ID
func (s SqliteService) Id() string {
    return SQLITE_SVC
}

// Db provides access to raw SqliteService db
// ISSUE: Original didn't protect against race conditions
func (s SqliteService) Db() *gorm.DB {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.db
}

// Configure sets up the service
// ISSUE: Original had minimal configuration and no validation
func (s *SqliteService) Configure(ctx *context.Context) error {
    s.config = DefaultConfig()
    s.config.Database = os.Getenv("DB_DATABASE")

    if s.config.Database == "" {
        return ErrDatabaseNotConfigured
    }

    return s.db.Configure(ctx)
}

// Start initializes the database connection and runs migrations
// ISSUE: Original lacked proper connection management and error handling
func (s *SqliteService) Start() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    config := &gorm.Config{
        Logger: logger.New(
            log.New(os.Stdout, "\r\n", log.LstdFlags),
            logger.Config{
                SlowThreshold:             time.Second,
                LogLevel:                  logger.Error,
                IgnoreRecordNotFoundError: true,
                Colorful:                  false,
            },
        ),
        NowFunc: func() time.Time {
            return time.Now().UTC()
        },
    }

    var err error
    s.db, err = gorm.Open(sqlite.Open(s.config.Database), config)
    if err != nil {
        return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
    }

    sqlDB, err := s.db.DB()
    if err != nil {
        return fmt.Errorf("failed to get database instance: %w", err)
    }

    // Configure connection pool
    sqlDB.SetMaxIdleConns(s.config.MaxIdleConns)
    sqlDB.SetMaxOpenConns(s.config.MaxOpenConns)
    sqlDB.SetConnMaxLifetime(s.config.MaxLifetime)

    // Run migrations
    if err := s.migrate(&nft_proxy.SolanaMedia{}); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    return nil
}

// CRUD Operations
// ISSUE: Original lacked context support and proper error handling

func (s *SqliteService) Find(ctx context.Context, out interface{}, where string, args ...interface{}) error {
    ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
    defer cancel()

    return s.handleError(s.db.WithContext(ctx).Find(out, where, args).Error)
}

func (s *SqliteService) Create(ctx context.Context, val interface{}) (interface{}, error) {
    return val, s.WithTx(ctx, func(tx *gorm.DB) error {
        return tx.Create(val).Error
    })
}

func (s *SqliteService) Update(ctx context.Context, old interface{}, new interface{}) (interface{}, error) {
    err := s.WithTx(ctx, func(tx *gorm.DB) error {
        return tx.Model(old).Updates(new).Error
    })
    return new, err
}

func (s *SqliteService) Delete(ctx context.Context, val interface{}) error {
    return s.WithTx(ctx, func(tx *gorm.DB) error {
        return tx.Delete(val).Error
    })
}

// BatchCreate handles bulk insertions
// ISSUE: Original lacked batch operation support
func (s *SqliteService) BatchCreate(ctx context.Context, values []interface{}, batchSize int) error {
    return s.WithTx(ctx, func(tx *gorm.DB) error {
        return tx.CreateInBatches(values, batchSize).Error
    })
}

// Transaction support
// ISSUE: Original lacked transaction support
func (s *SqliteService) WithTx(ctx context.Context, fn func(*gorm.DB) error) error {
    ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
    defer cancel()

    tx := s.db.WithContext(ctx).Begin()
    if tx.Error != nil {
        return fmt.Errorf("%w: %v", ErrTransactionFailed, tx.Error)
    }

    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
            panic(r)
        }
    }()

    if err := fn(tx); err != nil {
        tx.Rollback()
        return err
    }

    if err := tx.Commit().Error; err != nil {
        return fmt.Errorf("%w: %v", ErrTransactionFailed, err)
    }

    return nil
}

// Migration support
// ISSUE: Original had inconsistent error handling for migrations
func (s *SqliteService) migrate(values ...interface{}) error {
    if err := s.db.AutoMigrate(values...); err != nil {
        return fmt.Errorf("migration failed: %w", err)
    }
    return nil
}

// Health check
// ISSUE: Original lacked health checking capability
func (s *SqliteService) HealthCheck(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, time.Second*5)
    defer cancel()

    return s.db.WithContext(ctx).Raw("SELECT 1").Error
}

// Shutdown handles graceful shutdown
// ISSUE: Original had empty shutdown implementation
func (s *SqliteService) Shutdown(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.db != nil {
        sqlDB, err := s.db.DB()
        if err != nil {
            return fmt.Errorf("failed to get database instance: %w", err)
        }

        if err := sqlDB.Close(); err != nil {
            return fmt.Errorf("failed to close database connection: %w", err)
        }
    }
    return nil
}

// Error handling
// ISSUE: Original mixed HTTP status codes with database errors and lacked structured error handling
func (s *SqliteService) handleError(err error) error {
    if err == nil {
        return nil
    }

    switch {
    case errors.Is(err, gorm.ErrRecordNotFound):
        return &SqliteError{
            Code:    404,
            Message: "Record not found",
            Err:     err,
        }
    case errors.Is(err, context.DeadlineExceeded):
        return &SqliteError{
            Code:    504,
            Message: "Database timeout",
            Err:     err,
        }
    default:
        return &SqliteError{
            Code:    500,
            Message: "Database error",
            Err:     err,
        }
    }
}