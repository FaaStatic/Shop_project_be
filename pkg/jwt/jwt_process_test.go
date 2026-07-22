package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateTokenPair_ProducesDistinctAccessAndRefreshTokens(t *testing.T) {
	svc := NewJWTService("test-secret", 15, 24)

	pair, err := svc.GenerateTokenPair("user-1", "staff")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected both tokens to be non-empty")
	}
	if pair.AccessToken == pair.RefreshToken {
		t.Fatal("access and refresh tokens must be distinct (different jti/type/ttl)")
	}
	if pair.ExpiresIn != 15*60 {
		t.Errorf("ExpiresIn = %d, want %d (15 minutes in seconds)", pair.ExpiresIn, 15*60)
	}
}

func TestValidateToken_AccessTokenRoundTrip(t *testing.T) {
	svc := NewJWTService("test-secret", 15, 24)

	pair, err := svc.GenerateTokenPair("user-42", "superadmin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := svc.ValidateToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("expected access token to validate, got: %v", err)
	}
	if claims.UserID != "user-42" || claims.Role != "superadmin" || claims.Type != "access" {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestValidateToken_RefreshTokenHasRefreshType(t *testing.T) {
	svc := NewJWTService("test-secret", 15, 24)

	pair, err := svc.GenerateTokenPair("user-42", "staff")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := svc.ValidateToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("expected refresh token to validate, got: %v", err)
	}
	if claims.Type != "refresh" {
		t.Errorf("expected type=refresh, got %s", claims.Type)
	}
}

func TestValidateToken_RejectsExpiredToken(t *testing.T) {
	// A negative TTL (via accessMin=0 won't expire fast enough); build the
	// expired token directly using the same construction as generateToken but
	// with a TTL already in the past, so this test does not depend on real
	// clock sleeps.
	svc := NewJWTService("test-secret", 15, 24)
	expired, err := svc.generateToken("user-1", "staff", "access", -1*time.Minute)
	if err != nil {
		t.Fatalf("failed to build expired token fixture: %v", err)
	}

	if _, err := svc.ValidateToken(expired); err == nil {
		t.Fatal("expected an error validating an expired token")
	}
}

func TestValidateToken_RejectsWrongSecret(t *testing.T) {
	svc := NewJWTService("secret-a", 15, 24)
	other := NewJWTService("secret-b", 15, 24)

	pair, err := svc.GenerateTokenPair("user-1", "staff")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := other.ValidateToken(pair.AccessToken); err == nil {
		t.Fatal("expected validation to fail: token was signed with a different secret")
	}
}

func TestValidateToken_RejectsMalformedToken(t *testing.T) {
	svc := NewJWTService("test-secret", 15, 24)

	if _, err := svc.ValidateToken("not-a-jwt-token"); err == nil {
		t.Fatal("expected an error for a malformed token string")
	}
}

func TestValidateToken_RejectsUnexpectedSigningMethod(t *testing.T) {
	svc := NewJWTService("test-secret", 15, 24)

	// Craft a token using the "none" algorithm, which ValidateToken must
	// reject even though the header claims to be self-consistent — accepting
	// it would allow an attacker to forge tokens without knowing the secret.
	claims := Claims{
		UserID: "attacker",
		Role:   "superadmin",
		Type:   "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	unsigned, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to build 'none'-alg token fixture: %v", err)
	}

	if _, err := svc.ValidateToken(unsigned); err == nil {
		t.Fatal("expected ValidateToken to reject a token signed with alg=none")
	}
}

func TestValidateToken_RejectsHS256AlgConfusionWithNoneCheck(t *testing.T) {
	// Belt-and-suspenders: even a syntactically valid-looking HS-signed token
	// with a tampered payload (bit flip in the signature) must fail.
	svc := NewJWTService("test-secret", 15, 24)
	pair, err := svc.GenerateTokenPair("user-1", "staff")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	tampered := pair.AccessToken[:len(pair.AccessToken)-1] + "x"
	if strings.HasSuffix(pair.AccessToken, "x") {
		tampered = pair.AccessToken[:len(pair.AccessToken)-1] + "y"
	}

	if _, err := svc.ValidateToken(tampered); err == nil {
		t.Fatal("expected an error for a tampered signature")
	}
}
