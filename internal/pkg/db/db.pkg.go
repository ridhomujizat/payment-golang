package database

import (
	"fmt"
	"go-boilerplate/internal/pkg/redis"
	"time"

	"github.com/go-gorm/caches/v4"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	_logger "gorm.io/gorm/logger"
)

type Config struct {
	Host      string
	Port      int
	User      string
	Password  string
	Database  string
	SSLMode   string
	Driver    DriverEnum
	Cache     bool
	Rds       *redis.Client
	CacheTime time.Duration
}

type Database struct {
	*gorm.DB
	Config       *Config
	cursorCrypto *cursorCrypto
}

func Setup(cfg *Config) (*Database, error) {
	var db *gorm.DB
	var err error
	crypto, err := newCursorCrypto([]byte(cfg.User + cfg.Password + cfg.Database))
	if err != nil {
		return nil, err
	}
	// loc, err := time.LoadLocation("Asia/Jakarta")
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to load location: %w", err)
	// }

	gormConfig := &gorm.Config{
		Logger: _logger.Default.LogMode(_logger.Silent),
		// NowFunc: func() time.Time {
		// 	return time.Now().In(loc)
		// },
	}

	switch cfg.Driver {
	case POSTGRES:
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			cfg.Host,
			cfg.User,
			cfg.Password,
			cfg.Database,
			cfg.Port,
			cfg.SSLMode,
		)
		fmt.Println(dsn)
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)

	case MYSQL:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Database,
		)
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (supported: postgres, mysql)", cfg.Driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if cfg.Cache {
		var cachesPlugin *caches.Caches
		if cfg.Rds != nil && cfg.CacheTime > 0 {
			cachesPlugin = &caches.Caches{Conf: &caches.Config{
				Cacher: &redisCacher{
					rdb:       cfg.Rds.Client,
					cacheTime: cfg.CacheTime,
				},
			}}
		} else {
			cachesPlugin = &caches.Caches{Conf: &caches.Config{
				Cacher: &memoryCacher{},
			}}
		}

		_ = db.Use(cachesPlugin)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(20)

	return &Database{
		db,
		cfg,
		crypto,
	}, nil
}

func (db *Database) Migrate() error {
	err := db.AutoMigrate(
	/* Add your entities here
	 * &entity.Session{},
	 */
	)
	if err != nil {
		return fmt.Errorf("error migrating database: %w", err)

	}
	return nil
}

func (db *Database) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

func (db *Database) IsCloseConnection() bool {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return true
	}
	return sqlDB == nil || sqlDB.Stats().OpenConnections == 0
}
