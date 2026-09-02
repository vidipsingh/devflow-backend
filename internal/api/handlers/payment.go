package handlers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"
	"devflow-backend/internal/payment"
	"devflow-backend/internal/repository"
)

// rzClient is lazily initialised on first use so that godotenv.Load() in
// main.go's init() has already populated the env vars before we read them.
var (
	rzClientOnce sync.Once
	rzClient     *payment.Client
)

func getRzClient() *payment.Client {
	rzClientOnce.Do(func() {
		rzClient = payment.NewClient()
	})
	return rzClient
}

func getPaymentRepo() *repository.PaymentRepo {
	return repository.NewPaymentRepo(database.GetDB())
}

func getUserRepo() *repository.UserRepo {
	return repository.NewUserRepo(database.GetDB())
}

// POST /api/v1/payments/orders
func CreatePaymentOrder(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var body struct {
		PlanKey string `json:"planKey" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "planKey is required"})
		return
	}

	// Guard: use min(8, len) so we never panic on a short userID
	prefixLen := 8
	if len(userID) < prefixLen {
		prefixLen = len(userID)
	}
	receiptID := fmt.Sprintf("rcpt_%s_%d", userID[:prefixLen], time.Now().Unix())

	order, err := getRzClient().CreateOrder(userID, body.PlanKey, receiptID)
	if err != nil {
		// Use 502 so it's clear the error came from the upstream gateway
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// Persist an order record with status "created"
	payRepo := getPaymentRepo()
	p := &models.Payment{
		UserID:          userID,
		PlanKey:         body.PlanKey,
		RazorpayOrderID: order.OrderID,
		AmountPaise:     order.Amount,
		Currency:        order.Currency,
		Status:          "created",
	}
	if err := payRepo.Create(c, p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": order})
}

// POST /api/v1/payments/verify
func VerifyPayment(c *gin.Context) {
	userID := c.GetString("userID")

	var body struct {
		RazorpayOrderID   string `json:"razorpayOrderId"   binding:"required"`
		RazorpayPaymentID string `json:"razorpayPaymentId" binding:"required"`
		RazorpaySignature string `json:"razorpaySignature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Verify HMAC signature — prevents spoofing
	if !getRzClient().VerifySignature(body.RazorpayOrderID, body.RazorpayPaymentID, body.RazorpaySignature) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "invalid payment signature"})
		return
	}

	payRepo := getPaymentRepo()

	// 2. Look up the order we created
	order, err := payRepo.FindByOrderID(c, body.RazorpayOrderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	// 3. Ensure order belongs to this user
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "order does not belong to you"})
		return
	}

	// 4. Mark payment as paid
	if err := payRepo.MarkPaid(c, body.RazorpayOrderID, body.RazorpayPaymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update payment record"})
		return
	}

	// 5. Upgrade user plan
	userRepo := getUserRepo()
	if err := userRepo.UpdatePlan(c, userID, order.PlanKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upgrade plan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"plan":    order.PlanKey,
		"message": fmt.Sprintf("Successfully upgraded to %s plan", order.PlanKey),
	})
}

func GetPaymentHistory(c *gin.Context) {
	userID := c.GetString("userID")
	payRepo := getPaymentRepo()
	payments, err := payRepo.ListByUser(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payment history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": payments})
}
