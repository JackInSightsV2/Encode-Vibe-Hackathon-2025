package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OIDCValidator handles OpenID Connect ID token validation
type OIDCValidator struct {
	httpClient *http.Client
	jwksCache  map[string]*JWKSResponse
}

// JWKSResponse represents a JSON Web Key Set response
type JWKSResponse struct {
	Keys      []JWK     `json:"keys"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	X5c []string `json:"x5c,omitempty"`
	X5t string `json:"x5t,omitempty"`
}

// IDTokenClaims represents the claims in an OpenID Connect ID token
type IDTokenClaims struct {
	Issuer          string                 `json:"iss"`
	Subject         string                 `json:"sub"`
	Audience        interface{}            `json:"aud"`
	ExpiresAt       int64                  `json:"exp"`
	IssuedAt        int64                  `json:"iat"`
	AuthTime        int64                  `json:"auth_time,omitempty"`
	Nonce           string                 `json:"nonce,omitempty"`
	Email           string                 `json:"email,omitempty"`
	EmailVerified   bool                   `json:"email_verified,omitempty"`
	Name            string                 `json:"name,omitempty"`
	GivenName       string                 `json:"given_name,omitempty"`
	FamilyName      string                 `json:"family_name,omitempty"`
	Picture         string                 `json:"picture,omitempty"`
	Locale          string                 `json:"locale,omitempty"`
	Custom          map[string]interface{} `json:"-"`
	jwt.RegisteredClaims
}

// NewOIDCValidator creates a new OIDC validator
func NewOIDCValidator() *OIDCValidator {
	return &OIDCValidator{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		jwksCache: make(map[string]*JWKSResponse),
	}
}

// ValidateIDToken validates an OpenID Connect ID token
func (v *OIDCValidator) ValidateIDToken(provider *OAuth2Provider, idToken, nonce string) (*IDTokenClaims, error) {
	if idToken == "" {
		return nil, fmt.Errorf("empty ID token")
	}

	// Parse the token without verification first to get the header
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, &IDTokenClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	// Get the key ID from the token header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("no key ID found in token header")
	}

	// Get the public key for verification
	publicKey, err := v.getPublicKey(provider, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and validate the token with the public key
	claims := &IDTokenClaims{}
	validatedToken, err := jwt.ParseWithClaims(idToken, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !validatedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Validate claims
	if err := v.validateClaims(provider, claims, nonce); err != nil {
		return nil, fmt.Errorf("claims validation failed: %w", err)
	}

	return claims, nil
}

// getPublicKey retrieves the public key for token verification
func (v *OIDCValidator) getPublicKey(provider *OAuth2Provider, kid string) (*rsa.PublicKey, error) {
	if provider.JWKSEndpoint == "" {
		return nil, fmt.Errorf("no JWKS endpoint configured for provider")
	}

	// Check cache first
	cached, exists := v.jwksCache[provider.JWKSEndpoint]
	if exists && time.Now().Before(cached.ExpiresAt) {
		for _, key := range cached.Keys {
			if key.Kid == kid {
				return v.convertJWKToRSAPublicKey(&key)
			}
		}
	}

	// Fetch JWKS from the provider
	jwks, err := v.fetchJWKS(provider.JWKSEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Cache the JWKS
	v.jwksCache[provider.JWKSEndpoint] = &JWKSResponse{
		Keys:      jwks.Keys,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour), // Cache for 1 hour
	}

	// Find the key with matching kid
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return v.convertJWKToRSAPublicKey(&key)
		}
	}

	return nil, fmt.Errorf("key with ID %s not found", kid)
}

// fetchJWKS fetches the JSON Web Key Set from the provider
func (v *OIDCValidator) fetchJWKS(jwksURL string) (*JWKSResponse, error) {
	resp, err := v.httpClient.Get(jwksURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JWKS request failed: %d %s", resp.StatusCode, string(body))
	}

	var jwks JWKSResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, err
	}

	return &jwks, nil
}

// convertJWKToRSAPublicKey converts a JWK to an RSA public key
func (v *OIDCValidator) convertJWKToRSAPublicKey(jwk *JWK) (*rsa.PublicKey, error) {
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}

	// Decode the modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	
	// Convert exponent bytes to int
	var e int
	for _, b := range eBytes {
		e = e*256 + int(b)
	}

	// Create RSA public key
	publicKey := &rsa.PublicKey{
		N: n,
		E: e,
	}

	return publicKey, nil
}

// validateClaims validates the ID token claims
func (v *OIDCValidator) validateClaims(provider *OAuth2Provider, claims *IDTokenClaims, nonce string) error {
	now := time.Now()

	// Validate expiration
	if claims.ExpiresAt > 0 && now.Unix() > claims.ExpiresAt {
		return fmt.Errorf("token has expired")
	}

	// Validate issued at (allow some clock skew)
	if claims.IssuedAt > 0 && now.Unix() < (claims.IssuedAt-300) {
		return fmt.Errorf("token issued in the future")
	}

	// Validate issuer
	if provider.Issuer != "" && claims.Issuer != provider.Issuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", provider.Issuer, claims.Issuer)
	}

	// Validate audience
	if err := v.validateAudience(claims.Audience, provider.ClientID); err != nil {
		return err
	}

	// Validate nonce if provided
	if nonce != "" && claims.Nonce != nonce {
		return fmt.Errorf("invalid nonce: expected %s, got %s", nonce, claims.Nonce)
	}

	return nil
}

// validateAudience validates the audience claim
func (v *OIDCValidator) validateAudience(audience interface{}, clientID string) error {
	switch aud := audience.(type) {
	case string:
		if aud != clientID {
			return fmt.Errorf("invalid audience: expected %s, got %s", clientID, aud)
		}
	case []interface{}:
		found := false
		for _, a := range aud {
			if audStr, ok := a.(string); ok && audStr == clientID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("client ID not found in audience list")
		}
	default:
		return fmt.Errorf("invalid audience type")
	}

	return nil
}

// ClearCache clears the JWKS cache
func (v *OIDCValidator) ClearCache() {
	v.jwksCache = make(map[string]*JWKSResponse)
}

// GetCacheStatus returns the current cache status
func (v *OIDCValidator) GetCacheStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	for endpoint, cached := range v.jwksCache {
		status[endpoint] = map[string]interface{}{
			"cached_at":  cached.CachedAt,
			"expires_at": cached.ExpiresAt,
			"key_count":  len(cached.Keys),
			"expired":    time.Now().After(cached.ExpiresAt),
		}
	}

	return status
}

// CleanupExpiredCache removes expired entries from the cache
func (v *OIDCValidator) CleanupExpiredCache() {
	now := time.Now()
	for endpoint, cached := range v.jwksCache {
		if now.After(cached.ExpiresAt) {
			delete(v.jwksCache, endpoint)
		}
	}
}

// ValidateWithCustomClaims validates an ID token and extracts custom claims
func (v *OIDCValidator) ValidateWithCustomClaims(provider *OAuth2Provider, idToken, nonce string) (*IDTokenClaims, error) {
	claims, err := v.ValidateIDToken(provider, idToken, nonce)
	if err != nil {
		return nil, err
	}

	// Parse the token again to extract custom claims
	tokenParts := strings.Split(idToken, ".")
	if len(tokenParts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Decode the payload
	payload, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	// Parse all claims
	var allClaims map[string]interface{}
	if err := json.Unmarshal(payload, &allClaims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Extract custom claims (claims not in the standard set)
	standardClaims := map[string]bool{
		"iss": true, "sub": true, "aud": true, "exp": true, "iat": true,
		"auth_time": true, "nonce": true, "email": true, "email_verified": true,
		"name": true, "given_name": true, "family_name": true, "picture": true,
		"locale": true,
	}

	claims.Custom = make(map[string]interface{})
	for key, value := range allClaims {
		if !standardClaims[key] {
			claims.Custom[key] = value
		}
	}

	return claims, nil
}