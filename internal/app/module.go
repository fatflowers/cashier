package app

import (
    "github.com/fatflowers/cashier/internal/app/api/server"
    notificationhandler "github.com/fatflowers/cashier/internal/app/service/notification_handler"
    notificationlog "github.com/fatflowers/cashier/internal/app/service/notification_log"
    "github.com/fatflowers/cashier/internal/app/service/statistics"
    "github.com/fatflowers/cashier/internal/app/service/subscription"
    "github.com/fatflowers/cashier/internal/app/service/transaction"
    "github.com/fatflowers/cashier/internal/platform/db"
    "github.com/fatflowers/cashier/pkg/config"
    "github.com/fatflowers/cashier/pkg/logger"
	"time"

	"go.uber.org/fx"
)

const (
	DefaultStartTimeout = 15 * time.Second
	DefaultStopTimeout  = 10 * time.Second
)

var Module = fx.Options(
    logger.Module,
    config.Module,
    db.Module,
    server.Module,
    subscription.Module,
    statistics.Module,
    notificationlog.Module,
    notificationhandler.Module,
    transaction.Module,
)
