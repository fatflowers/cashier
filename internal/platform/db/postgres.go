package db

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/fatflowers/cashier/internal/models"
	cfgpkg "github.com/fatflowers/cashier/pkg/config"
	gormzap "github.com/fatflowers/cashier/pkg/gormlog"
)

func NewDB(l *zap.SugaredLogger, cfg *cfgpkg.Config) (*gorm.DB, error) {
	if cfg.Database.DSN == "" {
		l.Error("database DSN is empty")
		return nil, gorm.ErrInvalidDB
	}
	db, err := gorm.Open(postgres.Open(cfg.Database.DSN), &gorm.Config{Logger: gormzap.New(l)})
	if err != nil {
		l.Errorf("failed to connect database: %v", err)
		return nil, err
	}
	l.Infow("connected to postgres via DSN")
	return db, nil
}

var Module = fx.Options(
	fx.Provide(NewDB),
	fx.Invoke(AutoMigrate),
	fx.Invoke(registerDBClose),
)

// AutoMigrate runs GORM migrations on startup
func AutoMigrate(l *zap.SugaredLogger, db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Subscription{},
		&models.SubscriptionLog{},
		&models.SubscriptionDailySnapshot{},
		&models.Transaction{},
		&models.TransactionLog{},
		&models.PaymentNotificationLog{},
		&models.UserMembershipActiveItem{},
	); err != nil {
		l.Errorf("automigrate failed: %v", err)
		return err
	}
	l.Infow("automigrate completed")
	return nil
}

// registerDBClose ensures the underlying *sql.DB is closed on shutdown
func registerDBClose(lc fx.Lifecycle, l *zap.SugaredLogger, gdb *gorm.DB) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			sqlDB, err := gdb.DB()
			if err != nil {
				l.Warnw("gorm: get sql.DB failed", "err", err)
				return nil
			}
			l.Infow("closing postgres connection pool")
			return sqlDB.Close()
		},
	})
}
