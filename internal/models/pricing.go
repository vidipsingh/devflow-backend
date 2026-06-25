package models

type PricingPlan struct {
	PlanKey       string   `bson:"planKey"       json:"planKey"`
	Name          string   `bson:"name"          json:"name"`
	Price         float64  `bson:"price"         json:"price"`
	Currency      string   `bson:"currency"      json:"currency"`
	Period        string   `bson:"period"        json:"period"`
	Description   string   `bson:"description"   json:"description"`
	Badge         *string  `bson:"badge"         json:"badge"`
	IsHighlighted bool     `bson:"isHighlighted" json:"isHighlighted"`
	CtaLabel      string   `bson:"ctaLabel"      json:"ctaLabel"`
	CtaHref       string   `bson:"ctaHref"       json:"ctaHref"`
	Order         int      `bson:"order"         json:"order"`
	IsActive      bool     `bson:"isActive"      json:"isActive"`
	Features      []string `bson:"features"      json:"features"`
}