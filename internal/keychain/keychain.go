package keychain

// Credentials holds OAuth credentials for the Anthropic API.
type Credentials struct {
	AccessToken      string `json:"accessToken"`
	SubscriptionType string `json:"subscriptionType"`
}
