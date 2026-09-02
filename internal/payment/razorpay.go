package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"os"

	razorpay "github.com/razorpay/razorpay-go"
)

// PlanPrices maps planKey → amount in paise (1 INR = 100 paise).
// The Razorpay API expects amount as an integer (int, not int64).
var PlanPrices = map[string]int{
	"pro":  99900,  // ₹999/month
	"team": 299900, // ₹2999/month
}

type Client struct {
	rz     *razorpay.Client
	keyID  string
	secret string
}

func NewClient() *Client {
	keyID := os.Getenv("RAZORPAY_KEY_ID")
	secret := os.Getenv("RAZORPAY_KEY_SECRET")
	if keyID == "" || secret == "" {
		log.Println("[WARNING] RAZORPAY_KEY_ID or RAZORPAY_KEY_SECRET is not set — payment endpoints will fail")
	}
	return &Client{
		rz:     razorpay.NewClient(keyID, secret),
		keyID:  keyID,
		secret: secret,
	}
}

type OrderResponse struct {
	OrderID  string `json:"orderId"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	KeyID    string `json:"keyId"`
}

// CreateOrder creates a Razorpay order for the given plan.
func (c *Client) CreateOrder(userID, planKey, receiptID string) (*OrderResponse, error) {
	amount, ok := PlanPrices[planKey]
	if !ok {
		return nil, errors.New("invalid plan")
	}

	data := map[string]interface{}{
		// Razorpay SDK requires amount as int (paise), currency as string
		"amount":   amount,
		"currency": "INR",
		"receipt":  receiptID,
		"notes": map[string]interface{}{
			"userId":  userID,
			"planKey": planKey,
		},
	}

	body, err := c.rz.Order.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay create order: %w", err)
	}

	orderID, _ := body["id"].(string)

	// The SDK decodes JSON numbers as float64 (standard Go json.Unmarshal behaviour)
	amtFloat, _ := body["amount"].(float64)
	amtInt := int64(math.Round(amtFloat))

	return &OrderResponse{
		OrderID:  orderID,
		Amount:   amtInt,
		Currency: "INR",
		KeyID:    c.keyID,
	}, nil
}

// VerifySignature verifies the Razorpay payment signature.
// signature = HMAC-SHA256(keySecret, orderId + "|" + paymentId)
func (c *Client) VerifySignature(orderID, paymentID, signature string) bool {
	payload := orderID + "|" + paymentID
	mac := hmac.New(sha256.New, []byte(c.secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
