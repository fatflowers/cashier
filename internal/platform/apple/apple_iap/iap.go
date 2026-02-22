package apple_iap

import (
	"context"
	"errors"
	"fmt"

	"github.com/awa/go-iap/appstore"
	"github.com/awa/go-iap/appstore/api"
)

type GetAppleIAPClientOptions struct {
	KeyID        string
	KeyContent   string
	BundleID     string
	Issuer       string
	Sandbox      bool
	SharedSecret string
}

func GetAppleIAPClient(ctx context.Context, opts *GetAppleIAPClientOptions) (*api.StoreClient, error) {
	if opts == nil {
		return nil, errors.New("opts is nil")
	}

	c := &api.StoreConfig{
		KeyContent: []byte(opts.KeyContent),
		KeyID:      opts.KeyID,
		BundleID:   opts.BundleID,
		Issuer:     opts.Issuer,
		Sandbox:    opts.Sandbox,
	}

	return api.NewStoreClient(c), nil
}

type ReceiptInfo struct {
	OriginalPurchaseDateMs  string `json:"original_purchase_date_ms"`
	OriginalPurchaseDatePst string `json:"original_purchase_date_pst"`
	InAppOwnershipType      string `json:"in_app_ownership_type"`
	AppAccountToken         string `json:"app_account_token"`
	Quantity                string `json:"quantity"`
	ProductId               string `json:"product_id"`
	TransactionId           string `json:"transaction_id"`
	PurchaseDate            string `json:"purchase_date"`
	IsTrialPeriod           string `json:"is_trial_period"`
	OriginalTransactionId   string `json:"original_transaction_id"`
	PurchaseDateMs          string `json:"purchase_date_ms"`
	PurchaseDatePst         string `json:"purchase_date_pst"`
	OriginalPurchaseDate    string `json:"original_purchase_date"`
}

type Receipt struct {
	LatesetReceiptInfo []*ReceiptInfo `json:"latest_receipt_info"`
}

func VerifyServerVerificationData(ctx context.Context, receiptData string, opts *GetAppleIAPClientOptions) (*Receipt, error) {
	if opts == nil {
		return nil, errors.New("opts is nil")
	}

	client := appstore.New()
	if opts.Sandbox {
		client.ProductionURL = client.SandboxURL
	}

	var result Receipt

	err := client.Verify(ctx, appstore.IAPRequest{
		ReceiptData:            receiptData,
		Password:               opts.SharedSecret,
		ExcludeOldTransactions: true,
	}, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to verify receipt: %w", err)
	}

	return &result, nil
}
