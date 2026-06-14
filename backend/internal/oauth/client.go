package oauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const clientAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

type JWK struct {
	Crv string `json:"crv"`
	Kty string `json:"kty"`
	D   string `json:"d,omitempty"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type AuthServerMetadata struct {
	Issuer                             string `json:"issuer"`
	AuthorizationEndpoint              string `json:"authorization_endpoint"`
	TokenEndpoint                      string `json:"token_endpoint"`
	PushedAuthorizationRequestEndpoint string `json:"pushed_authorization_request_endpoint"`
}

type PARResponse struct {
	RequestURI string `json:"request_uri"`
	ExpiresIn  int    `json:"expires_in"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	Sub          string `json:"sub"`
}

type Client struct {
	ClientID    string
	RedirectURI string

	signingKey *ecdsa.PrivateKey
	signingKid string
	http       *http.Client
}

func NewClient(clientID, redirectURI string, signingKey *ecdsa.PrivateKey) *Client {
	return &Client{
		ClientID:    clientID,
		RedirectURI: redirectURI,
		signingKey:  signingKey,
		signingKid:  jwkThumbprintP256(&signingKey.PublicKey),
		http:        &http.Client{Timeout: 15 * time.Second, CheckRedirect: checkRedirect},
	}
}

func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func coord(i *big.Int) string { return b64u(i.FillBytes(make([]byte, 32))) }

func GenerateDPoPKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func GenerateSigningKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

const SigningKeyKV = "oauth_signing_key"

type KVStore interface {
	GetOrCreateEncryptedKV(ctx context.Context, key, value string) (string, error)
}

func LoadSigningKey(ctx context.Context, kv KVStore) (*ecdsa.PrivateKey, error) {
	candidate, err := GenerateSigningKey()
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(PrivateJWK(candidate))
	if err != nil {
		return nil, err
	}
	stored, err := kv.GetOrCreateEncryptedKV(ctx, SigningKeyKV, string(raw))
	if err != nil {
		return nil, err
	}
	var j JWK
	if err := json.Unmarshal([]byte(stored), &j); err != nil {
		return nil, err
	}
	return ParsePrivateJWK(j.Crv, j.D, j.X, j.Y)
}

func publicJWK(pub *ecdsa.PublicKey) JWK {
	return JWK{Crv: "P-256", Kty: "EC", X: coord(pub.X), Y: coord(pub.Y)}
}

func PrivateJWK(priv *ecdsa.PrivateKey) JWK {
	j := publicJWK(&priv.PublicKey)
	j.D = coord(priv.D)
	return j
}

func ParsePrivateJWK(crv, d, x, y string) (*ecdsa.PrivateKey, error) {
	if crv != "" && crv != "P-256" {
		return nil, fmt.Errorf("unsupported curve %q", crv)
	}
	db, err := base64.RawURLEncoding.DecodeString(d)
	if err != nil {
		return nil, fmt.Errorf("decode d: %w", err)
	}
	xb, err := base64.RawURLEncoding.DecodeString(x)
	if err != nil {
		return nil, fmt.Errorf("decode x: %w", err)
	}
	yb, err := base64.RawURLEncoding.DecodeString(y)
	if err != nil {
		return nil, fmt.Errorf("decode y: %w", err)
	}
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(xb), Y: new(big.Int).SetBytes(yb)},
		D:         new(big.Int).SetBytes(db),
	}, nil
}

func jwkThumbprintP256(pub *ecdsa.PublicKey) string {
	json := `{"crv":"P-256","kty":"EC","x":"` + coord(pub.X) + `","y":"` + coord(pub.Y) + `"}`
	sum := sha256.Sum256([]byte(json))
	return b64u(sum[:])
}

func signJWT(header, claims map[string]any, key *ecdsa.PrivateKey) (string, error) {
	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	cb, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	input := b64u(hb) + "." + b64u(cb)
	sum := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, key, sum[:])
	if err != nil {
		return "", err
	}
	var rb, sb [32]byte
	r.FillBytes(rb[:])
	s.FillBytes(sb[:])
	return input + "." + b64u(append(rb[:], sb[:]...)), nil
}

func (c *Client) dpopProof(key *ecdsa.PrivateKey, method, uri, nonce, ath string) (string, error) {
	now := time.Now()
	jti := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, jti); err != nil {
		return "", err
	}
	header := map[string]any{
		"typ": "dpop+jwt",
		"alg": "ES256",
		"jwk": publicJWK(&key.PublicKey),
	}
	claims := map[string]any{
		"jti": b64u(jti),
		"htm": method,
		"htu": uri,
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	if ath != "" {
		claims["ath"] = ath
	}
	return signJWT(header, claims, key)
}

func (c *Client) clientAssertion(issuer string) (string, error) {
	now := time.Now()
	jti := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, jti); err != nil {
		return "", err
	}
	header := map[string]any{"alg": "ES256", "kid": c.signingKid}
	claims := map[string]any{
		"iss": c.ClientID,
		"sub": c.ClientID,
		"aud": issuer,
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"jti": b64u(jti),
	}
	return signJWT(header, claims, c.signingKey)
}

func GeneratePKCE() (verifier, challenge string) {
	b := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, b)
	verifier = b64u(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge = b64u(sum[:])
	return
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, b)
	return b64u(b)
}

func (c *Client) PublicJWKS() map[string]any {
	pub := publicJWK(&c.signingKey.PublicKey)
	return map[string]any{
		"keys": []any{
			map[string]any{
				"kty": "EC", "crv": "P-256", "x": pub.X, "y": pub.Y,
				"use": "sig", "alg": "ES256", "kid": c.signingKid,
			},
		},
	}
}

func (c *Client) ResolveHandle(ctx context.Context, handle string) (string, error) {
	handle = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(handle)), "@")
	if did, err := c.resolveHandleAt(ctx, "https://public.api.bsky.app", handle); err == nil && strings.HasPrefix(did, "did:") {
		return did, nil
	}
	candidates := []string{"https://" + handle}
	if parts := strings.Split(handle, "."); len(parts) > 2 {
		candidates = append(candidates, "https://"+strings.Join(parts[1:], "."))
	}
	for _, svc := range candidates {
		if did, err := c.resolveHandleAt(ctx, svc, handle); err == nil && strings.HasPrefix(did, "did:") {
			return did, nil
		}
	}
	return "", fmt.Errorf("could not resolve handle %q", handle)
}

func (c *Client) resolveHandleAt(ctx context.Context, service, handle string) (string, error) {
	service = strings.TrimRight(service, "/")
	if service != "https://public.api.bsky.app" {
		if err := validatePDSURL(service); err != nil {
			return "", err
		}
	}
	u := service + "/xrpc/com.atproto.identity.resolveHandle?handle=" + url.QueryEscape(handle)
	var out struct {
		DID string `json:"did"`
	}
	if err := c.getJSON(ctx, u, &out); err != nil {
		return "", err
	}
	return out.DID, nil
}

func (c *Client) ResolveDIDToPDS(ctx context.Context, did string) (string, error) {
	var docURL string
	switch {
	case strings.HasPrefix(did, "did:plc:"):
		docURL = "https://plc.directory/" + did
	case strings.HasPrefix(did, "did:web:"):
		domain := strings.TrimPrefix(did, "did:web:")
		host := domain
		if i := strings.IndexByte(host, ':'); i >= 0 {
			host = host[:i]
		}
		if err := validatePDSURL("https://" + host); err != nil {
			return "", err
		}
		docURL = "https://" + strings.ReplaceAll(domain, ":", "/") + "/.well-known/did.json"
	default:
		return "", fmt.Errorf("unsupported DID method: %s", did)
	}
	var doc struct {
		Service []struct {
			Type            string `json:"type"`
			ServiceEndpoint string `json:"serviceEndpoint"`
		} `json:"service"`
	}
	if err := c.getJSON(ctx, docURL, &doc); err != nil {
		return "", err
	}
	for _, svc := range doc.Service {
		if svc.Type == "AtprotoPersonalDataServer" {
			if err := validatePDSURL(svc.ServiceEndpoint); err != nil {
				return "", err
			}
			return strings.TrimRight(svc.ServiceEndpoint, "/"), nil
		}
	}
	return "", fmt.Errorf("no PDS found in DID document")
}

func (c *Client) ResolveDIDToHandle(ctx context.Context, did string) (string, error) {
	var docURL string
	switch {
	case strings.HasPrefix(did, "did:plc:"):
		docURL = "https://plc.directory/" + did
	case strings.HasPrefix(did, "did:web:"):
		domain := strings.TrimPrefix(did, "did:web:")
		host := domain
		if i := strings.IndexByte(host, ':'); i >= 0 {
			host = host[:i]
		}
		if err := validatePDSURL("https://" + host); err != nil {
			return "", err
		}
		docURL = "https://" + strings.ReplaceAll(domain, ":", "/") + "/.well-known/did.json"
	default:
		return "", fmt.Errorf("unsupported DID method: %s", did)
	}
	var doc struct {
		AlsoKnownAs []string `json:"alsoKnownAs"`
	}
	if err := c.getJSON(ctx, docURL, &doc); err != nil {
		return "", err
	}
	for _, aka := range doc.AlsoKnownAs {
		if handle := strings.TrimPrefix(aka, "at://"); handle != aka {
			return handle, nil
		}
	}
	return "", fmt.Errorf("no handle found in DID document")
}

func (c *Client) GetAuthServerMetadata(ctx context.Context, pdsURL string) (*AuthServerMetadata, error) {
	if err := validatePDSURL(pdsURL); err != nil {
		return nil, err
	}
	var resource struct {
		AuthorizationServers []string `json:"authorization_servers"`
	}
	if err := c.getJSON(ctx, strings.TrimRight(pdsURL, "/")+"/.well-known/oauth-protected-resource", &resource); err != nil {
		return nil, err
	}
	if len(resource.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("no authorization servers found")
	}
	authServer := resource.AuthorizationServers[0]
	if err := validatePDSURL(authServer); err != nil {
		return nil, err
	}
	return c.fetchAuthMeta(ctx, authServer)
}

func (c *Client) GetAuthServerMetadataForSignup(ctx context.Context, rawURL string) (*AuthServerMetadata, error) {
	if err := validatePDSURL(rawURL); err != nil {
		return nil, err
	}
	if meta, err := c.fetchAuthMeta(ctx, rawURL); err == nil {
		return meta, nil
	}
	return c.GetAuthServerMetadata(ctx, rawURL)
}

func (c *Client) fetchAuthMeta(ctx context.Context, authServer string) (*AuthServerMetadata, error) {
	var meta AuthServerMetadata
	if err := c.getJSON(ctx, strings.TrimRight(authServer, "/")+"/.well-known/oauth-authorization-server", &meta); err != nil {
		return nil, err
	}
	if meta.Issuer == "" || meta.TokenEndpoint == "" || meta.AuthorizationEndpoint == "" || meta.PushedAuthorizationRequestEndpoint == "" {
		return nil, fmt.Errorf("incomplete auth server metadata")
	}
	return &meta, nil
}

func (c *Client) getJSON(ctx context.Context, u string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %d", u, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (c *Client) SendPAR(ctx context.Context, meta *AuthServerMetadata, loginHint, scope string, dpopKey *ecdsa.PrivateKey, challenge string) (*PARResponse, string, string, error) {
	return c.SendPARWithPrompt(ctx, meta, loginHint, scope, dpopKey, challenge, "")
}

func (c *Client) SendPARWithPrompt(ctx context.Context, meta *AuthServerMetadata, loginHint, scope string, dpopKey *ecdsa.PrivateKey, challenge, prompt string) (*PARResponse, string, string, error) {
	state := randomState()
	assertion, err := c.clientAssertion(meta.Issuer)
	if err != nil {
		return nil, "", "", err
	}
	form := url.Values{}
	form.Set("client_id", c.ClientID)
	form.Set("redirect_uri", c.RedirectURI)
	form.Set("response_type", "code")
	form.Set("scope", scope)
	form.Set("state", state)
	form.Set("code_challenge", challenge)
	form.Set("code_challenge_method", "S256")
	form.Set("client_assertion_type", clientAssertionType)
	form.Set("client_assertion", assertion)
	if loginHint != "" {
		form.Set("login_hint", loginHint)
	}
	if prompt != "" {
		form.Set("prompt", prompt)
	}
	body, nonce, err := c.formPOST(ctx, meta.PushedAuthorizationRequestEndpoint, form, dpopKey, "", "")
	if err != nil {
		return nil, "", nonce, err
	}
	var par PARResponse
	if err := json.Unmarshal(body, &par); err != nil {
		return nil, "", nonce, err
	}
	return &par, state, nonce, nil
}

func (c *Client) AuthorizeURL(meta *AuthServerMetadata, requestURI string) string {
	u, _ := url.Parse(meta.AuthorizationEndpoint)
	q := u.Query()
	q.Set("client_id", c.ClientID)
	q.Set("request_uri", requestURI)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) ExchangeCode(ctx context.Context, meta *AuthServerMetadata, code, verifier string, dpopKey *ecdsa.PrivateKey, initialNonce string) (*TokenResponse, error) {
	assertion, err := c.clientAssertion(meta.Issuer)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.RedirectURI)
	form.Set("client_id", c.ClientID)
	form.Set("code_verifier", verifier)
	form.Set("client_assertion_type", clientAssertionType)
	form.Set("client_assertion", assertion)
	body, _, err := c.formPOST(ctx, meta.TokenEndpoint, form, dpopKey, "", initialNonce)
	if err != nil {
		return nil, err
	}
	var tok TokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func (c *Client) RefreshToken(ctx context.Context, meta *AuthServerMetadata, refreshToken string, dpopKey *ecdsa.PrivateKey) (*TokenResponse, error) {
	assertion, err := c.clientAssertion(meta.Issuer)
	if err != nil {
		return nil, err
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", c.ClientID)
	form.Set("client_assertion_type", clientAssertionType)
	form.Set("client_assertion", assertion)
	body, _, err := c.formPOST(ctx, meta.TokenEndpoint, form, dpopKey, "", "")
	if err != nil {
		return nil, err
	}
	var tok TokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

func (c *Client) formPOST(ctx context.Context, endpoint string, form url.Values, dpopKey *ecdsa.PrivateKey, ath, initialNonce string) ([]byte, string, error) {
	nonce := initialNonce
	encoded := form.Encode()
	for attempt := 0; attempt < 2; attempt++ {
		proof, err := c.dpopProof(dpopKey, http.MethodPost, endpoint, nonce, ath)
		if err != nil {
			return nil, nonce, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(encoded))
		if err != nil {
			return nil, nonce, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("DPoP", proof)
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, nonce, err
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if n := resp.Header.Get("DPoP-Nonce"); n != "" {
			nonce = n
		}
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			return body, nonce, nil
		}
		if attempt == 0 && nonce != "" && strings.Contains(string(body), "use_dpop_nonce") {
			continue
		}
		return nil, nonce, fmt.Errorf("oauth %s: %d: %s", endpoint, resp.StatusCode, string(body))
	}
	return nil, nonce, fmt.Errorf("oauth %s: failed after nonce retry", endpoint)
}
