package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"devflow-backend/internal/models"
	"devflow-backend/internal/payment"
	"devflow-backend/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const platformFeeRate = 0.20

// Snippets

func GetTopSnippets(ctx context.Context, limit int, sortBy string) ([]models.Snippet, error) {
	if limit <= 0 || limit > 50 {
		limit = 6
	}
	return repository.FindPublishedSnippets(ctx, limit, sortBy)
}

func GetSnippetByID(ctx context.Context, id string) (*models.Snippet, error) {
	s, err := repository.FindSnippetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, errors.New("snippet not found")
	}
	go repository.IncrementSnippetViews(context.Background(), id)
	return s, nil
}

func GetMySnippets(ctx context.Context, creatorID string) ([]models.Snippet, error) {
	return repository.FindSnippetsByCreator(ctx, creatorID)
}


type CreateSnippetInput struct {
	Title       string
	Description string
	Code        string
	Preview     string
	Language    string
	Tags        []string
	Category    string
	Version     string
	PricingType string  // "free" | "paid"
	Price       float64 // only for paid, in INR
	Currency    string  // default "INR"
}

func CreateSnippet(ctx context.Context, creatorID, creatorUsername string, in CreateSnippetInput) (*models.Snippet, error) {
	if in.Title == "" || in.Code == "" {
		return nil, errors.New("title and code are required")
	}
	if in.PricingType == "paid" && in.Price <= 0 {
		return nil, errors.New("price must be > 0 for paid snippets")
	}
	if in.PricingType == "" {
		in.PricingType = "free"
	}
	if in.Currency == "" {
		in.Currency = "INR"
	}
	coid, err := bson.ObjectIDFromHex(creatorID)
	if err != nil {
		return nil, errors.New("invalid creator id")
	}

	s := &models.Snippet{
		CreatorID:       coid,
		CreatorUsername: creatorUsername,
		Title:           in.Title,
		Description:     in.Description,
		Code:            in.Code,
		Preview:         in.Preview,
		Language:        in.Language,
		Tags:            in.Tags,
		Category:        in.Category,
		Status:          "draft",
		Version:         in.Version,
		Pricing: models.SnippetPricing{
			Type:     in.PricingType,
			Price:    in.Price,
			Currency: in.Currency,
		},
	}
	if err := repository.InsertSnippet(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

type UpdateSnippetInput struct {
	Title       *string
	Description *string
	Code        *string
	Preview     *string
	Language    *string
	Tags        []string
	Category    *string
	Version     *string
	PricingType *string
	Price       *float64
}

func UpdateSnippet(ctx context.Context, id, ownerID string, in UpdateSnippetInput) error {
	update := bson.M{}
	if in.Title != nil       { update["title"] = *in.Title }
	if in.Description != nil { update["description"] = *in.Description }
	if in.Code != nil        { update["code"] = *in.Code }
	if in.Preview != nil     { update["preview"] = *in.Preview }
	if in.Language != nil    { update["language"] = *in.Language }
	if in.Tags != nil        { update["tags"] = in.Tags }
	if in.Category != nil    { update["category"] = *in.Category }
	if in.Version != nil     { update["version"] = *in.Version }
	if in.PricingType != nil { update["pricing.type"] = *in.PricingType }
	if in.Price != nil       { update["pricing.price"] = *in.Price }

	if len(update) == 0 {
		return errors.New("nothing to update")
	}
	return repository.UpdateSnippet(ctx, id, ownerID, update)
}

func PublishSnippet(ctx context.Context, id, ownerID string) error {
	return repository.PublishSnippet(ctx, id, ownerID)
}

func UnpublishSnippet(ctx context.Context, id, ownerID string) error {
	return repository.UnpublishSnippet(ctx, id, ownerID)
}

func DeleteSnippet(ctx context.Context, id, ownerID string) error {
	return repository.DeleteSnippet(ctx, id, ownerID)
}

// Download

// CanDownload returns true when the user is allowed to download the snippet.
func CanDownload(ctx context.Context, snippetID, userID string) (bool, *models.Snippet, error) {
	s, err := repository.FindSnippetByID(ctx, snippetID)
	if err != nil || s == nil {
		return false, nil, errors.New("snippet not found")
	}
	if s.Status != "published" {
		return false, nil, errors.New("snippet is not published")
	}
	if s.Pricing.Type == "free" {
		return true, s, nil
	}
	// Paid: check ownership first (creator can always download their own)
	if s.CreatorID.Hex() == userID {
		return true, s, nil
	}
	purchased, err := repository.HasPurchased(ctx, snippetID, userID)
	return purchased, s, err
}

// RecordDownload increments the download counter.
func RecordDownload(ctx context.Context, snippetID string) {
	go repository.IncrementSnippetDownloads(context.Background(), snippetID)
}

// Purchases

type PurchaseOrderResponse struct {
	OrderID  string `json:"orderId"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	KeyID    string `json:"keyId"`
}

func CreatePurchaseOrder(ctx context.Context, snippetID, buyerID string) (*PurchaseOrderResponse, error) {
	s, err := repository.FindSnippetByID(ctx, snippetID)
	if err != nil || s == nil {
		return nil, errors.New("snippet not found")
	}
	if s.Status != "published" {
		return nil, errors.New("snippet is not published")
	}
	if s.Pricing.Type != "paid" {
		return nil, errors.New("snippet is free — no purchase needed")
	}
	if s.CreatorID.Hex() == buyerID {
		return nil, errors.New("you cannot purchase your own snippet")
	}

	// Idempotency: if already purchased, refuse
	already, err := repository.HasPurchased(ctx, snippetID, buyerID)
	if err != nil {
		return nil, err
	}
	if already {
		return nil, errors.New("already purchased")
	}

	amountPaise := int64(math.Round(s.Pricing.Price * 100))
	// Always use a hard 3-char ISO currency code — Razorpay rejects anything else.
	currency := "INR"
	if len(s.Pricing.Currency) == 3 {
		currency = s.Pricing.Currency
	}

	prefixLen := 8
	if len(buyerID) < prefixLen {
		prefixLen = len(buyerID)
	}
	receiptID := fmt.Sprintf("snip_%s_%d", buyerID[:prefixLen], time.Now().Unix())

	rzClient := payment.NewClient()
	order, err := rzClient.CreateOrderRaw(int(amountPaise), currency, receiptID)
	if err != nil {
		return nil, fmt.Errorf("razorpay: %w", err)
	}

	boid, _ := bson.ObjectIDFromHex(buyerID)
	purchase := &models.SnippetPurchase{
		SnippetID:       s.ID,
		BuyerID:         boid,
		SellerID:        s.CreatorID,
		AmountPaise:     amountPaise,
		Currency:        currency,
		RazorpayOrderID: order.OrderID,
		Status:          "created",
	}
	if err := repository.InsertPurchase(ctx, purchase); err != nil {
		return nil, err
	}

	return &PurchaseOrderResponse{
		OrderID:  order.OrderID,
		Amount:   order.Amount,
		Currency: order.Currency,
		KeyID:    os.Getenv("RAZORPAY_KEY_ID"),
	}, nil
}

func VerifyPurchase(ctx context.Context, orderID, paymentID, signature, buyerID string) error {
	rzClient := payment.NewClient()
	if !rzClient.VerifySignature(orderID, paymentID, signature) {
		return errors.New("invalid payment signature")
	}
	
	purchase, err := repository.FindPurchaseByOrderID(ctx, orderID)
	if err != nil || purchase == nil {
		return errors.New("purchase order not found")
	}
	if purchase.BuyerID.Hex() != buyerID {
		return errors.New("purchase does not belong to you")
	}
	if purchase.Status == "paid" {
		return nil
	}

	if err := repository.MarkPurchasePaid(ctx, orderID, paymentID); err != nil {
		return err
	}

	// Update snippet earnings + purchase count atomically
	go func() {
		bgCtx := context.Background()
		fee := float64(purchase.AmountPaise) / 100 * platformFeeRate
		earnings := float64(purchase.AmountPaise)/100 - fee
		_ = repository.UpdateSnippet(bgCtx, purchase.SnippetID.Hex(), purchase.SellerID.Hex(), bson.M{
			"earnings.totalRevenue":    earnings + fee,
			"earnings.creatorEarnings": earnings,
			"earnings.platformFee":     fee,
		})
		_, _ = repository.IncrementSnippetPurchaseCount(bgCtx, purchase.SnippetID.Hex())
	}()

	return nil
}

func GetMyPurchases(ctx context.Context, buyerID string) ([]models.SnippetPurchase, error) {
	return repository.ListPurchasesByBuyer(ctx, buyerID)
}

// Reviews

func GetReviews(ctx context.Context, snippetID string) ([]models.SnippetReview, error) {
	return repository.FindReviewsBySnippet(ctx, snippetID)
}

type CreateReviewInput struct {
	Rating  int
	Comment string
}

func CreateReview(ctx context.Context, snippetID, authorID, authorUsername string, in CreateReviewInput) (*models.SnippetReview, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	s, err := repository.FindSnippetByID(ctx, snippetID)
	if err != nil || s == nil {
		return nil, errors.New("snippet not found")
	}
	if s.Status != "published" {
		return nil, errors.New("snippet is not published")
	}
	// Must have purchased
	if s.CreatorID.Hex() == authorID {
		return nil, errors.New("you cannot review your own snippet")
	}

	// Paid snippets: must have purchased before reviewing
	if s.Pricing.Type == "paid" {
		purchased, err := repository.HasPurchased(ctx, snippetID, authorID)
		if err != nil {
			return nil, err
		}
		if !purchased {
			return nil, errors.New("purchase the snippet before reviewing")
		}
	}

	// Idempotency: one review per user per snippet
	already, err := repository.HasReviewed(ctx, snippetID, authorID)
	if err != nil {
		return nil, err
	}
	if already {
		return nil, errors.New("you have already reviewed this snippet")
	}

	soid, _ := bson.ObjectIDFromHex(snippetID)
	aoid, _ := bson.ObjectIDFromHex(authorID)
	r := &models.SnippetReview{
		SnippetID:      soid,
		AuthorID:       aoid,
		AuthorUsername: authorUsername,
		Rating:         in.Rating,
		Comment:        in.Comment,
	}
	if err := repository.InsertReview(ctx, r); err != nil {
		return nil, err
	}

	// Recompute average in background
	go repository.RecalcSnippetRating(context.Background(), snippetID)

	return r, nil
}

