package notification_log

import (
	"context"
	"github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/logctx"
	"github.com/fatflowers/cashier/pkg/tool"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Service struct {
	db  *gorm.DB
	log *zap.SugaredLogger
}

func New(db *gorm.DB, log *zap.SugaredLogger) *Service { return &Service{db: db, log: log} }

// Save asynchronously persists a payment notification log. Nil input is ignored.
func (s *Service) Save(ctx context.Context, log *models.PaymentNotificationLog) {
	go func() {
		if log == nil {
			return
		}
		if log.ID == "" {
			log.ID = tool.GenerateUUIDV7()
		}
		if err := s.db.Save(log).Error; err != nil {
			logctx.FromCtx(ctx, s.log).Errorf("failed to save notification log: %v", err)
		}
	}()
}
