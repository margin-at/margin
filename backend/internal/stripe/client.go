package stripe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"margin.at/internal/logger"
)

const apiBase = "https://api.stripe.com/v1"

type Client struct {
	secretKey     string
	webhookSecret string
	priceMonthly  string
	priceYearly   string
	httpClient    *http.Client
}

func NewClient() *Client {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		logger.Info("STRIPE_SECRET_KEY not set — billing disabled")
	}
	return &Client{
		secretKey:     key,
		webhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		priceMonthly:  os.Getenv("STRIPE_PRICE_ID_MONTHLY"),
		priceYearly:   os.Getenv("STRIPE_PRICE_ID_YEARLY"),
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) IsEnabled() bool {
	return c.secretKey != ""
}

func (c *Client) PriceID(plan string) string {
	if plan == "yearly" {
		return c.priceYearly
	}
	return c.priceMonthly
}

type CheckoutSession struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}

func (c *Client) CreateCheckoutSession(customerID, priceID, successURL, cancelURL string) (*CheckoutSession, error) {
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("line_items[0][price]", priceID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	if customerID != "" {
		form.Set("customer", customerID)
	}
	form.Set("customer_email", "")
	form.Del("customer_email")

	var result struct {
		URL string `json:"url"`
		ID  string `json:"id"`
	}
	if err := c.post("/checkout/sessions", form, &result); err != nil {
		return nil, err
	}
	return &CheckoutSession{URL: result.URL, ID: result.ID}, nil
}

func (c *Client) CreatePortalSession(customerID, returnUrl string) (string, error) {
	form := url.Values{}
	form.Set("customer", customerID)
	form.Set("return_url", returnUrl)

	var result struct {
		URL string `json:"url"`
	}
	if err := c.post("/billing_portal/sessions", form, &result); err != nil {
		return "", err
	}
	return result.URL, nil
}

func (c *Client) CreateCustomer(did string) (string, error) {
	form := url.Values{}
	form.Set("metadata[did]", did)

	var result struct {
		ID string `json:"id"`
	}
	if err := c.post("/customers", form, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

type Subscription struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	Customer          string `json:"customer"`
	CurrentPeriodEnd  int64  `json:"current_period_end"`
	Plan struct {
		ID    string `json:"id"`
	} `json:"plan"`
}

func (c *Client) GetSubscription(subID string) (*Subscription, error) {
	var result Subscription
	if err := c.get("/subscriptions/"+subID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type WebhookEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type webhookObject struct {
	Object json.RawMessage `json:"object"`
}

func (e *WebhookEvent) Object() json.RawMessage {
	var obj webhookObject
	json.Unmarshal(e.Data, &obj)
	return obj.Object
}

func (e *WebhookEvent) GetString(path string) string {
	var obj map[string]interface{}
	json.Unmarshal(e.Object(), &obj)
	for _, key := range splitPath(path) {
		if m, ok := obj[key].(map[string]interface{}); ok {
			obj = m
		} else {
			if s, ok := obj[key].(string); ok {
				return s
			}
			return ""
		}
	}
	return ""
}

func (e *WebhookEvent) GetInt64(path string) int64 {
	var obj map[string]interface{}
	json.Unmarshal(e.Object(), &obj)
	for _, key := range splitPath(path) {
		if m, ok := obj[key].(map[string]interface{}); ok {
			obj = m
		} else {
			if f, ok := obj[key].(float64); ok {
				return int64(f)
			}
			return 0
		}
	}
	return 0
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func (c *Client) ConstructWebhookEvent(payload []byte, signature string) (*WebhookEvent, error) {
	if c.webhookSecret == "" {
		return nil, fmt.Errorf("STRIPE_WEBHOOK_SECRET not set")
	}

	timestamp, sig, err := parseSignature(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature header: %w", err)
	}

	if time.Since(time.Unix(timestamp, 0)) > 5*time.Minute {
		return nil, fmt.Errorf("timestamp outside tolerance")
	}

	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, fmt.Errorf("signature mismatch")
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

func parseSignature(header string) (int64, string, error) {
	var timestamp int64
	var sig string
	parts := splitAndTrim(header, ",")
	for _, p := range parts {
		kv := splitAndTrim(p, "=")
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
		var err error
		timestamp, err = strconv.ParseInt(kv[1], 10, 64)
			if err != nil {
				return 0, "", fmt.Errorf("invalid timestamp: %w", err)
			}
		case "v1":
			sig = kv[1]
		}
	}
	if timestamp == 0 || sig == "" {
		return 0, "", fmt.Errorf("missing t or v1 in signature")
	}
	return timestamp, sig, nil
}

func splitAndTrim(s, sep string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if string(c) == sep {
			trimmed := trimSpaces(current)
			if trimmed != "" {
				result = append(result, trimmed)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	trimmed := trimSpaces(current)
	if trimmed != "" {
		result = append(result, trimmed)
	}
	return result
}

func trimSpaces(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", apiBase+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *Client) post(path string, form url.Values, result interface{}) error {
	req, err := http.NewRequest("POST", apiBase+path, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req, result)
}

func (c *Client) do(req *http.Request, result interface{}) error {
	req.SetBasicAuth(c.secretKey, "")
	req.Header.Set("Stripe-Version", "2024-06-20")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("stripe request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		json.Unmarshal(body, &errResp)
		msg := errResp.Error.Message
		if msg == "" {
			msg = string(body)
		}
		return fmt.Errorf("stripe API %d: %s", resp.StatusCode, msg)
	}

	if result != nil {
		return json.Unmarshal(body, result)
	}
	return nil
}
