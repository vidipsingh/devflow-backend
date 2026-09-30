package handlers

import (
	"log"
	"net/http"
	"strconv"

	api "devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// Helpers

func snippetID(c *gin.Context) string      { return c.Param("snippetId") }
func callerID(c *gin.Context) string       { return c.GetString("userID") }
func callerUsername(c *gin.Context) string { return c.GetString("username") }

// Public handlers

// GET /api/v1/marketplace/snippets?limit=6&sort=downloads
func GetSnippets(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "6"))
	sortBy := c.DefaultQuery("sort", "downloads")

	snippets, err := service.GetTopSnippets(c.Request.Context(), limit, sortBy)
	if err != nil {
		api.InternalError(c, "failed to fetch snippets")
		return
	}
	if snippets == nil {
		snippets = []models.Snippet{}
	}
	api.OK(c, snippets)
}

// GET /api/v1/marketplace/snippets/:snippetId
func GetSnippet(c *gin.Context) {
	s, err := service.GetSnippetByID(c.Request.Context(), snippetID(c))
	if err != nil {
		api.NotFound(c, err.Error())
		return
	}
	api.OK(c, s)
}

// GET /api/v1/marketplace/snippets/:snippetId/reviews
func GetSnippetReviews(c *gin.Context) {
	reviews, err := service.GetReviews(c.Request.Context(), snippetID(c))
	if err != nil {
		api.InternalError(c, "failed to fetch reviews")
		return
	}
	api.OK(c, reviews)
}

// Authenticated handlers

// GET /api/v1/marketplace/my-snippets
func GetMySnippets(c *gin.Context) {
	snippets, err := service.GetMySnippets(c.Request.Context(), callerID(c))
	if err != nil {
		api.InternalError(c, "failed to fetch snippets")
		return
	}
	api.OK(c, snippets)
}

// POST /api/v1/marketplace/snippets
func CreateSnippet(c *gin.Context) {
	var body struct {
		Title       string   `json:"title"       binding:"required"`
		Description string   `json:"description"`
		Code        string   `json:"code"        binding:"required"`
		Preview     string   `json:"preview"`
		Language    string   `json:"language"`
		Tags        []string `json:"tags"`
		Category    string   `json:"category"`
		Version     string   `json:"version"`
		PricingType string   `json:"pricingType"`
		Price       float64  `json:"price"`
		Currency    string   `json:"currency"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.BadRequest(c, err.Error())
		return
	}

	s, err := service.CreateSnippet(c.Request.Context(), callerID(c), callerUsername(c),
		service.CreateSnippetInput{
			Title:       body.Title,
			Description: body.Description,
			Code:        body.Code,
			Preview:     body.Preview,
			Language:    body.Language,
			Tags:        body.Tags,
			Category:    body.Category,
			Version:     body.Version,
			PricingType: body.PricingType,
			Price:       body.Price,
			Currency:    body.Currency,
		})
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}
	api.Created(c, s)
}

// PATCH /api/v1/marketplace/snippets/:snippetId
func UpdateSnippet(c *gin.Context) {
	var body struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description"`
		Code        *string  `json:"code"`
		Preview     *string  `json:"preview"`
		Language    *string  `json:"language"`
		Tags        []string `json:"tags"`
		Category    *string  `json:"category"`
		Version     *string  `json:"version"`
		PricingType *string  `json:"pricingType"`
		Price       *float64 `json:"price"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.BadRequest(c, err.Error())
		return
	}
	err := service.UpdateSnippet(c.Request.Context(), snippetID(c), callerID(c),
		service.UpdateSnippetInput{
			Title:       body.Title,
			Description: body.Description,
			Code:        body.Code,
			Preview:     body.Preview,
			Language:    body.Language,
			Tags:        body.Tags,
			Category:    body.Category,
			Version:     body.Version,
			PricingType: body.PricingType,
			Price:       body.Price,
		})
	if err != nil {
		if err.Error() == "snippet not found or not yours" {
			api.NotFound(c, err.Error())
			return
		}
		api.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// PATCH /api/v1/marketplace/snippets/:snippetId/publish
func PublishSnippet(c *gin.Context) {
	action := c.DefaultQuery("action", "publish")
	var err error
	if action == "unpublish" {
		err = service.UnpublishSnippet(c.Request.Context(), snippetID(c), callerID(c))
	} else {
		err = service.PublishSnippet(c.Request.Context(), snippetID(c), callerID(c))
	}
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "action": action})
}

// DELETE /api/v1/marketplace/snippets/:snippetId
func DeleteSnippet(c *gin.Context) {
	if err := service.DeleteSnippet(c.Request.Context(), snippetID(c), callerID(c)); err != nil {
		api.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GET /api/v1/marketplace/snippets/:snippetId/download
func DownloadSnippet(c *gin.Context) {
	allowed, s, err := service.CanDownload(c.Request.Context(), snippetID(c), callerID(c))
	if err != nil {
		api.NotFound(c, err.Error())
		return
	}
	if !allowed {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"success": false,
			"error":   "purchase this snippet to download it",
		})
		return
	}
	service.RecordDownload(c.Request.Context(), snippetID(c))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"code":     s.Code,
			"language": s.Language,
			"title":    s.Title,
			"version":  s.Version,
		},
	})
}

// Purchase handlers

// POST /api/v1/marketplace/snippets/:snippetId/purchase/order
func CreatePurchaseOrder(c *gin.Context) {
	resp, err := service.CreatePurchaseOrder(c.Request.Context(), snippetID(c), callerID(c))
	if err != nil {
		switch err.Error() {
		case "snippet not found", "snippet is not published":
			api.NotFound(c, err.Error())
		case "already purchased":
			c.JSON(http.StatusConflict, gin.H{"success": false, "error": err.Error()})
		default:
			api.BadRequest(c, err.Error())
		}
		return
	}
	api.OK(c, resp)
}

// POST /api/v1/marketplace/snippets/:snippetId/purchase/verify
func VerifyPurchase(c *gin.Context) {
	buyer := callerID(c)
	sID := snippetID(c)
	log.Printf("[marketplace] VerifyPurchase start — snippetId=%s buyerId=%s", sID, buyer)

	var body struct {
		RazorpayOrderID   string `json:"razorpayOrderId"   binding:"required"`
		RazorpayPaymentID string `json:"razorpayPaymentId" binding:"required"`
		RazorpaySignature string `json:"razorpaySignature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Printf("[marketplace] VerifyPurchase bad request — %v", err)
		api.BadRequest(c, err.Error())
		return
	}
	log.Printf("[marketplace] VerifyPurchase verifying — orderId=%s paymentId=%s", body.RazorpayOrderID, body.RazorpayPaymentID)

	err := service.VerifyPurchase(c.Request.Context(),
		body.RazorpayOrderID, body.RazorpayPaymentID, body.RazorpaySignature, buyer)
	if err != nil {
		log.Printf("[marketplace] VerifyPurchase error — %v", err)
		if err.Error() == "invalid payment signature" {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": err.Error()})
			return
		}
		api.BadRequest(c, err.Error())
		return
	}
	log.Printf("[marketplace] VerifyPurchase success — snippetId=%s buyerId=%s orderId=%s", sID, buyer, body.RazorpayOrderID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "purchase verified"})
}

// GET /api/v1/marketplace/purchases
func GetMyPurchases(c *gin.Context) {
	purchases, err := service.GetMyPurchases(c.Request.Context(), callerID(c))
	if err != nil {
		api.InternalError(c, "failed to fetch purchases")
		return
	}
	api.OK(c, purchases)
}

// Review handlers

// POST /api/v1/marketplace/snippets/:snippetId/reviews
func CreateReview(c *gin.Context) {
	var body struct {
		Rating  int    `json:"rating"  binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.BadRequest(c, err.Error())
		return
	}
	r, err := service.CreateReview(c.Request.Context(),
		snippetID(c), callerID(c), callerUsername(c),
		service.CreateReviewInput{Rating: body.Rating, Comment: body.Comment})
	if err != nil {
		api.BadRequest(c, err.Error())
		return
	}
	api.Created(c, r)
}
