package statistics

import (
	"context"
	"fmt"
	"github.com/fatflowers/cashier/internal/models"
	"github.com/fatflowers/cashier/pkg/tool"
	"github.com/fatflowers/cashier/pkg/types"
	"sync"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Statistic types mirrored from the legacy implementation
type StatisticType string

const (
	// Daily counts and GMV
	StatisticTypeDailyTransactionCount StatisticType = "daily_transaction_count"
	StatisticTypeDailyGmv              StatisticType = "daily_gmv"
	StatisticTypeTotalGmv              StatisticType = "total_gmv"

	// Membership (subscription) related
	StatisticTypeDailyMembershipCount            StatisticType = "daily_membership_count"
	StatisticTypeDailyNewMembershipCount         StatisticType = "daily_new_membership_count"
	StatisticTypeTotalMembershipCount            StatisticType = "total_membership_count"
	StatisticTypeDailyAccumulatedMembershipCount StatisticType = "daily_accumulated_membership_count"

	// Renewal metrics
	StatisticTypeRenewalSuccessRate StatisticType = "renewal_success_rate"
)

// Filter types supported by certain statistic types
type MembershipStatisticFilterType string

const (
	MembershipStatisticFilterTypeIsFirstPurchase MembershipStatisticFilterType = "is_first_purchase"
	MembershipStatisticFilterTypeIsAutoRenew     MembershipStatisticFilterType = "is_auto_renew"
	MembershipStatisticFilterTypePaymentItemID   MembershipStatisticFilterType = "payment_item_id"
)

var filterTypes = []MembershipStatisticFilterType{
	MembershipStatisticFilterTypeIsFirstPurchase,
	MembershipStatisticFilterTypeIsAutoRenew,
	MembershipStatisticFilterTypePaymentItemID,
}

var validFilters = map[MembershipStatisticFilterType][]StatisticType{
	MembershipStatisticFilterTypeIsFirstPurchase: {StatisticTypeDailyTransactionCount, StatisticTypeDailyGmv},
	MembershipStatisticFilterTypeIsAutoRenew:     {StatisticTypeDailyTransactionCount, StatisticTypeDailyGmv},
	MembershipStatisticFilterTypePaymentItemID:   {StatisticTypeDailyTransactionCount, StatisticTypeDailyGmv},
}

type MembershipStatisticDataItem struct {
	ID StatisticType `json:"id"`
}

type MembershipStatisticRequest struct {
	Filters   []*types.CommonFilter          `json:"filters"`
	DataItems []*MembershipStatisticDataItem `json:"data_items"`
}

func (f *MembershipStatisticRequest) GetFilters(statisticType StatisticType) *MembershipStatisticRequest {
	if f == nil || len(f.Filters) == 0 {
		return f
	}
	var result MembershipStatisticRequest
	for _, filter := range f.Filters {
		if statisticTypes, ok := validFilters[MembershipStatisticFilterType(filter.Field)]; ok {
			if lo.Contains(statisticTypes, statisticType) {
				result.Filters = append(result.Filters, filter)
			}
		} else {
			result.Filters = append(result.Filters, filter)
		}
	}
	return &result
}

// Build composes a WHERE clause based on provided filters, with custom handling for
// special filter fields like is_first_purchase and is_auto_renew.
func (f *MembershipStatisticRequest) Build(builder clause.Builder) {
	if len(f.Filters) == 0 {
		builder.WriteString("1=1")
		return
	}
	for i, filter := range f.Filters {
		if i > 0 {
			builder.WriteString(" AND ")
		}
		switch filter.Field {
		case string(MembershipStatisticFilterTypeIsFirstPurchase):
			if len(filter.Values) > 0 && fmt.Sprint(filter.Values[0]) == "true" {
				builder.WriteString("extra->>'is_first_purchase' = 'true'")
			} else {
				builder.WriteString("(extra->>'is_first_purchase' = 'false' OR extra->>'is_first_purchase' IS NULL)")
			}
		case string(MembershipStatisticFilterTypeIsAutoRenew):
			if len(filter.Values) > 0 && fmt.Sprint(filter.Values[0]) == "true" {
				builder.WriteString("parent_transaction_id is not null and parent_transaction_id != transaction_id")
			} else {
				builder.WriteString("(parent_transaction_id is null or parent_transaction_id = transaction_id)")
			}
		default:
			filter.Build(builder)
		}
	}
}

type MembershipStatisticResponseDataItem struct {
	Date   string `json:"date"`
	Label  string `json:"label,omitempty"`
	Value  int64  `json:"value"`
	Value2 int64  `json:"value2,omitempty"`
	Value3 int64  `json:"value3,omitempty"`
}

type MembershipStatisticResponse struct {
	DataItems map[StatisticType][]MembershipStatisticResponseDataItem `json:"data_items"`
}

// Service provides statistics operations
type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service { return &Service{db: db} }

// SaveSubscriptionDailySnapshot persists a daily snapshot of a user's subscription state
func (s *Service) SaveSubscriptionDailySnapshot(ctx context.Context, subscription *models.Subscription, snapshotDate time.Time) error {
	if subscription == nil {
		return fmt.Errorf("nil subscription")
	}
	snap := &models.SubscriptionDailySnapshot{
		ID:                tool.GenerateUUIDV7(),
		UserID:            subscription.UserID,
		Status:            subscription.Status,
		NextAutoRenewAt:   subscription.NextAutoRenewAt,
		ExpireAt:          subscription.ExpireAt,
		Extra:             subscription.Extra,
		CreatedAt:         subscription.CreatedAt,
		UpdatedAt:         subscription.UpdatedAt,
		SnapshotDate:      snapshotDate.Format(time.DateOnly),
		SnapshotCreatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Create(snap).Error
}

// Internal helpers for various stats
func (s *Service) getDailyTransactionCount(ctx context.Context, request *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	q := s.db.WithContext(ctx).Table("transaction").
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, count(*) as value").
		Where("provider_id != ?", types.PaymentProviderInner).
		Where(clause.Where{Exprs: []clause.Expression{request.GetFilters(StatisticTypeDailyTransactionCount)}}).
		Group("TO_CHAR(created_at, 'YYYY-MM-DD')").
		Order("date")
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getDailyGmv(ctx context.Context, request *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	q := s.db.WithContext(ctx).Table("transaction").
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, currency AS label, sum(price) as value").
		Where("provider_id != ?", types.PaymentProviderInner).
		Where(clause.Where{Exprs: []clause.Expression{request.GetFilters(StatisticTypeDailyGmv)}}).
		Group("TO_CHAR(created_at, 'YYYY-MM-DD')").
		Group("currency").
		Order(clause.OrderByColumn{Column: clause.Column{Name: "date"}, Desc: true})
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getTotalGmv(ctx context.Context, _ *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	err := s.db.WithContext(ctx).Raw(`
WITH min_max_dates AS (
    SELECT MIN(DATE(created_at)) as min_date, MAX(DATE(created_at)) as max_date
    FROM transaction
),
distinct_dates AS (
    SELECT generate_series(min_date, max_date, '1 day'::interval) as date FROM min_max_dates
),
dates AS (
    SELECT TO_CHAR(date, 'YYYY-MM-DD') as date FROM distinct_dates
),
currencies AS (
    SELECT DISTINCT currency as label FROM transaction WHERE provider_id != ?
),
date_currency_combinations AS (
    SELECT d.date, c.label FROM dates d CROSS JOIN currencies c
),
gmv_date AS (
    SELECT dc.date, dc.label, COALESCE(SUM(t.price), 0) as value
    FROM date_currency_combinations dc
    LEFT JOIN transaction t 
      ON TO_CHAR(t.created_at, 'YYYY-MM-DD') = dc.date 
     AND t.currency = dc.label 
     AND t.provider_id != ?
    GROUP BY dc.date, dc.label
)
SELECT d.date as date, d.label as label, SUM(s.value) as value
FROM gmv_date d
LEFT JOIN gmv_date s ON s.date <= d.date AND s.label = d.label
GROUP BY d.date, d.label
ORDER BY d.date DESC, d.label ASC
`, types.PaymentProviderInner, types.PaymentProviderInner).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getDailyMembershipCount(ctx context.Context, request *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	q := s.db.WithContext(ctx).Table((models.SubscriptionDailySnapshot{}).TableName()).
		Select("snapshot_date as date, count(*) as value").
		Where(clause.Where{Exprs: []clause.Expression{request.GetFilters(StatisticTypeDailyMembershipCount)}}).
		Group("snapshot_date").
		Order("snapshot_date")
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getDailyNewMembershipCount(ctx context.Context, _ *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	err := s.db.WithContext(ctx).Raw(`
WITH distinct_dates AS (
    SELECT DISTINCT DATE(created_at) as date FROM subscription ORDER BY date
),
user_id_date AS (
    SELECT user_id, DATE(created_at) as date FROM subscription
)
SELECT d.date, COUNT(DISTINCT s.user_id) as value
FROM distinct_dates d
JOIN user_id_date s ON s.date = d.date
GROUP BY d.date
ORDER BY d.date DESC
`).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getTotalMembershipCount(ctx context.Context, request *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	q := s.db.WithContext(ctx).Table((models.Subscription{}).TableName()).
		Select("count(*) as value").
		Where(clause.Where{Exprs: []clause.Expression{request.GetFilters(StatisticTypeTotalMembershipCount)}}).
		Where("status = ?", types.SubscriptionStatusActive).
		Where("expire_at >= ?", time.Now())
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getDailyAccumulatedMembershipCount(ctx context.Context, _ *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	err := s.db.WithContext(ctx).Raw(`
WITH min_max_dates AS (
    SELECT MIN(DATE(created_at)) as min_date, MAX(DATE(created_at)) as max_date FROM subscription
),
distinct_dates AS (
    SELECT generate_series(min_date, max_date, '1 day'::interval) as date FROM min_max_dates
),
user_id_date AS (
    SELECT user_id, DATE(created_at) as date FROM subscription
)
SELECT TO_CHAR(d.date, 'YYYY-MM-DD') as date, COUNT(DISTINCT s.user_id) as value
FROM distinct_dates d
LEFT JOIN user_id_date s ON s.date <= d.date
GROUP BY d.date
ORDER BY d.date DESC
`).Scan(&results).Error
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getRenewalSuccessRate(ctx context.Context, _ *MembershipStatisticRequest) ([]MembershipStatisticResponseDataItem, error) {
	var results []MembershipStatisticResponseDataItem
	sql := `
WITH renewal_count AS (
  SELECT user_id, DATE(purchase_at) as purchase_date, DATE(next_auto_renew_at) as next_auto_renew_date
  FROM transaction
  WHERE provider_id!='inner'
    AND parent_transaction_id IS NOT NULL
  GROUP BY user_id, DATE(purchase_at), DATE(next_auto_renew_at)
),
successful_renewals AS (
  SELECT r1.next_auto_renew_date, COUNT(*) as count1
  FROM renewal_count r1
  JOIN renewal_count r2 ON r1.user_id = r2.user_id AND r1.next_auto_renew_date = r2.purchase_date
  GROUP BY r1.next_auto_renew_date
),
total_renewals AS (
  SELECT next_auto_renew_date, COUNT(*) as count2
  FROM renewal_count
  WHERE next_auto_renew_date IS NOT NULL
    AND next_auto_renew_date < DATE(NOW() + INTERVAL '1 day')
  GROUP BY next_auto_renew_date
)
SELECT 
  TO_CHAR(COALESCE(s.next_auto_renew_date, t.next_auto_renew_date), 'YYYY-MM-DD') as date,
  CASE WHEN t.count2 = 0 THEN 0
       ELSE CAST(ROUND(LEAST(COALESCE(s.count1, 0) * 100.0 / t.count2, 100), 2) * 100 AS INTEGER)
  END as value,
  t.count2 as value2,
  COALESCE(s.count1, 0) as value3
FROM total_renewals t
LEFT JOIN successful_renewals s ON t.next_auto_renew_date = s.next_auto_renew_date
ORDER BY date DESC`
	if err := s.db.WithContext(ctx).Raw(sql).Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Service) getMembershipStatistic(ctx context.Context, request *MembershipStatisticRequest, dataItem *MembershipStatisticDataItem) ([]MembershipStatisticResponseDataItem, error) {
	switch dataItem.ID {
	case StatisticTypeDailyTransactionCount:
		return s.getDailyTransactionCount(ctx, request)
	case StatisticTypeDailyGmv:
		return s.getDailyGmv(ctx, request)
	case StatisticTypeTotalGmv:
		return s.getTotalGmv(ctx, request)
	case StatisticTypeDailyMembershipCount:
		return s.getDailyMembershipCount(ctx, request)
	case StatisticTypeDailyNewMembershipCount:
		return s.getDailyNewMembershipCount(ctx, request)
	case StatisticTypeTotalMembershipCount:
		return s.getTotalMembershipCount(ctx, request)
	case StatisticTypeDailyAccumulatedMembershipCount:
		return s.getDailyAccumulatedMembershipCount(ctx, request)
	case StatisticTypeRenewalSuccessRate:
		return s.getRenewalSuccessRate(ctx, request)
	default:
		return nil, fmt.Errorf("invalid data item id: %s", dataItem.ID)
	}
}

func (s *Service) GetDailyMembershipStatistic(ctx context.Context, request *MembershipStatisticRequest) (*MembershipStatisticResponse, error) {
	var wg sync.WaitGroup
	errChan := make(chan error, len(request.DataItems))
	resChan := make(chan *lo.Entry[StatisticType, []MembershipStatisticResponseDataItem], len(request.DataItems))

	for _, item := range request.DataItems {
		wg.Add(1)
		go func(di *MembershipStatisticDataItem) {
			defer wg.Done()
			// check filter applicability
			for _, filter := range request.Filters {
				ft := MembershipStatisticFilterType(filter.Field)
				if lo.Contains(filterTypes, ft) && !lo.Contains(validFilters[ft], di.ID) {
					resChan <- &lo.Entry[StatisticType, []MembershipStatisticResponseDataItem]{Key: di.ID, Value: nil}
					return
				}
			}
			res, err := s.getMembershipStatistic(ctx, request, di)
			if err != nil {
				errChan <- err
				return
			}
			resChan <- &lo.Entry[StatisticType, []MembershipStatisticResponseDataItem]{Key: di.ID, Value: res}
		}(item)
	}

	go func() { wg.Wait(); close(errChan); close(resChan) }()

	results := make(map[StatisticType][]MembershipStatisticResponseDataItem)
	for i := 0; i < len(request.DataItems); i++ {
		select {
		case err := <-errChan:
			if err != nil {
				return nil, err
			}
		case entry := <-resChan:
			results[entry.Key] = entry.Value
		}
	}
	return &MembershipStatisticResponse{DataItems: results}, nil
}
