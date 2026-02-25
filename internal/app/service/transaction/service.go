package transaction

import (
	"context"
	"fmt"
	subscription "github.com/fatflowers/cashier/internal/app/service/subscription"
	models "github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/config"
	types "github.com/fatflowers/cashier/pkg/types"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	cfg                     *config.Config
	log                     *zap.SugaredLogger
	appleTransactionManager *AppleTransactionManager
	subSvc                  *subscription.Service
	db                      *gorm.DB
}

func NewService(cfg *config.Config, log *zap.SugaredLogger, apple *AppleTransactionManager, sub *subscription.Service, db *gorm.DB) TransactionManager {
	return &Service{cfg: cfg, log: log, appleTransactionManager: apple, subSvc: sub, db: db}
}

func (s *Service) VerifyTransaction(ctx context.Context, req *TransactionVerifyRequest) (*VerifyTransactionResult, error) {
	switch req.ProviderID {
	case string(types.PaymentProviderApple):
		return s.appleTransactionManager.VerifyTransaction(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.ProviderID)
	}
}

func (s *Service) ParseVerificationData(ctx context.Context, req *VerificationDataRequest) (*VerifiedData, error) {
	switch req.ProviderID {
	case string(types.PaymentProviderApple):
		return s.appleTransactionManager.ParseVerificationData(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", req.ProviderID)
	}
}

func (s *Service) RefundTransaction(ctx context.Context, transactionId string, outRefundId string) error {
	// For Apple, not supported currently
	return fmt.Errorf("refund not supported for provider")
}

// filtersAnd is a helper to combine multiple CommonFilter into a single clause.Expression
type filtersAnd struct{ filters []*types.CommonFilter }

func (w filtersAnd) Build(builder clause.Builder) {
	if len(w.filters) == 0 {
		builder.WriteString("1=1")
		return
	}
	exprs := make([]clause.Expression, 0, len(w.filters))
	for _, f := range w.filters {
		exprs = append(exprs, f)
	}
	clause.And(exprs...).Build(builder)
}

// ScanTransactions implements paginated/admin listing with filters
func (s *Service) ScanTransactions(ctx context.Context, req *ScanTransactionsRequest) (*ScanTransactionsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.From < 0 {
		req.From = 0
	}

	tx := s.db.WithContext(ctx).Model(&models.Transaction{})
	if len(req.Filters) > 0 {
		tx = tx.Where(clause.Where{Exprs: []clause.Expression{filtersAnd{filters: req.Filters}}})
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count transactions: %w", err)
	}

	var rows []*models.Transaction

	q := tx.Limit(req.Size)

	if req.From > 0 {
		q = q.Offset(req.From)
	}

	if req.SortBy != "" {
		q = q.Order(clause.OrderBy{Columns: []clause.OrderByColumn{{Column: clause.Column{Name: req.SortBy}, Desc: req.SortOrder != "asc"}}})
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}

	return &ScanTransactionsResponse{Items: rows, Total: total}, nil
}
