package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"margin.at/internal/logger"
)

type Client struct {
	apiToken    string
	accountID   string
	namespaceID string
	rootDomain  string
	httpClient  *http.Client
}

func NewClient() *Client {
	token := os.Getenv("CF_API_TOKEN")
	if token == "" {
		logger.Info("CF_API_TOKEN not set — custom domain management disabled")
	}
	return &Client{
		apiToken:    token,
		accountID:   os.Getenv("CF_ACCOUNT_ID"),
		namespaceID: os.Getenv("CF_KV_NAMESPACE_ID"),
		rootDomain:  os.Getenv("CF_ROOT_DOMAIN"),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) IsEnabled() bool {
	return c.apiToken != "" && c.rootDomain != ""
}

func (c *Client) RootDomain() string {
	return c.rootDomain
}

func (c *Client) do(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBody)
	}
	req, err := http.NewRequest(method, "https://api.cloudflare.com/client/v4"+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

type apiResponse struct {
	Result  json.RawMessage `json:"result"`
	Success bool            `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (c *Client) GetZoneID(domain string) (string, error) {
	resp, err := c.do("GET", "/zones?name="+domain, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Result []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	if len(res.Result) == 0 {
		return "", fmt.Errorf("zone not found for %s", domain)
	}
	return res.Result[0].ID, nil
}

type CustomHostnameResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	SSL    struct {
		Status            string `json:"status"`
		ValidationRecords []struct {
			TxtName  string `json:"txt_name"`
			TxtValue string `json:"txt_value"`
			HTTPUrl  string `json:"http_url"`
			HTTPBody string `json:"http_body"`
		} `json:"validation_records"`
	} `json:"ssl"`
	OwnershipVerification struct {
		Type  string `json:"type"`
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"ownership_verification"`
	OwnershipVerificationHTTP struct {
		HTTPUrl  string `json:"http_url"`
		HTTPBody string `json:"http_body"`
	} `json:"ownership_verification_http"`
}

func (c *Client) CreateCustomHostname(zoneID, hostname string) (*CustomHostnameResult, error) {
	body := map[string]interface{}{
		"hostname": hostname,
		"ssl": map[string]interface{}{
			"method": "txt",
			"type":   "dv",
			"settings": map[string]interface{}{
				"min_tls_version": "1.2",
				"http2":           "on",
			},
		},
	}

	resp, err := c.do("POST", fmt.Sprintf("/zones/%s/custom_hostnames", zoneID), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res apiResponse
	json.NewDecoder(resp.Body).Decode(&res)
	if !res.Success {
		msg := "unknown"
		if len(res.Errors) > 0 {
			msg = res.Errors[0].Message
		}
		return nil, fmt.Errorf("create_custom_hostname failed: %s", msg)
	}

	var result CustomHostnameResult
	json.Unmarshal(res.Result, &result)
	return &result, nil
}

func (c *Client) GetCustomHostname(zoneID, id string) (*CustomHostnameResult, error) {
	resp, err := c.do("GET", fmt.Sprintf("/zones/%s/custom_hostnames/%s", zoneID, id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res apiResponse
	json.NewDecoder(resp.Body).Decode(&res)

	var result CustomHostnameResult
	json.Unmarshal(res.Result, &result)
	return &result, nil
}

func (c *Client) DeleteCustomHostname(zoneID, id string) error {
	resp, err := c.do("DELETE", fmt.Sprintf("/zones/%s/custom_hostnames/%s", zoneID, id), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) UpdateFallbackOrigin(zoneID, origin string) error {
	body := map[string]string{"origin": origin}
	resp, err := c.do("PUT", fmt.Sprintf("/zones/%s/custom_hostnames/fallback_origin", zoneID), body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) EnsureFallbackOrigin(zoneID string) error {
	if c.rootDomain == "" {
		return nil
	}
	fallbackOrigin := fmt.Sprintf("reading.%s", c.rootDomain)
	return c.UpdateFallbackOrigin(zoneID, fallbackOrigin)
}

func (c *Client) KVPut(key, value string) error {
	encodedKey := url.PathEscape(key)
	path := fmt.Sprintf("/accounts/%s/storage/kv/namespaces/%s/values/%s",
		c.accountID, c.namespaceID, encodedKey)

	req, _ := http.NewRequest("PUT", "https://api.cloudflare.com/client/v4"+path, bytes.NewBuffer([]byte(value)))
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("kv_put failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) KVDelete(key string) error {
	encodedKey := url.PathEscape(key)
	path := fmt.Sprintf("/accounts/%s/storage/kv/namespaces/%s/values/%s",
		c.accountID, c.namespaceID, encodedKey)

	req, _ := http.NewRequest("DELETE", "https://api.cloudflare.com/client/v4"+path, nil)
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) EnsureRouting(hostname, handle string) error {
	return c.KVPut("host:"+hostname, handle)
}

func (c *Client) RemoveRouting(hostname string) error {
	return c.KVDelete("host:" + hostname)
}

type VerificationRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	HTTPUrl  string `json:"httpUrl,omitempty"`
	HTTPBody string `json:"httpBody,omitempty"`
}

func ExtractVerificationRecords(r *CustomHostnameResult) []VerificationRecord {
	var records []VerificationRecord

	if r.OwnershipVerification.Type != "" && r.OwnershipVerification.Name != "" && r.OwnershipVerification.Value != "" {
		records = append(records, VerificationRecord{
			Type:  r.OwnershipVerification.Type,
			Name:  r.OwnershipVerification.Name,
			Value: r.OwnershipVerification.Value,
		})
	}

	if r.OwnershipVerificationHTTP.HTTPUrl != "" && r.OwnershipVerificationHTTP.HTTPBody != "" {
		records = append(records, VerificationRecord{
			Type:     "http",
			HTTPUrl:  r.OwnershipVerificationHTTP.HTTPUrl,
			HTTPBody: r.OwnershipVerificationHTTP.HTTPBody,
		})
	}

	for _, rec := range r.SSL.ValidationRecords {
		if rec.TxtName != "" && rec.TxtValue != "" {
			records = append(records, VerificationRecord{
				Type:  "ssl_txt",
				Name:  rec.TxtName,
				Value: rec.TxtValue,
			})
		}
		if rec.HTTPUrl != "" && rec.HTTPBody != "" {
			records = append(records, VerificationRecord{
				Type:     "ssl_http",
				HTTPUrl:  rec.HTTPUrl,
				HTTPBody: rec.HTTPBody,
			})
		}
	}

	return records
}

func CombinedStatus(r *CustomHostnameResult) string {
	status := r.Status
	sslStatus := r.SSL.Status
	if status == "active" && sslStatus == "active" {
		return "active"
	}
	if status == "active" || sslStatus == "active" {
		return "pending_ssl"
	}
	return "pending"
}
