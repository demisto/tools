package domain

// Rule for BlueCoat
type Rule struct {
	Category string `json:"category"` // Category to place the rule
	URL      string `json:"url"`      // URL of the rule
}

func (r *Rule) Key() string {
	return r.Category + r.URL
}
