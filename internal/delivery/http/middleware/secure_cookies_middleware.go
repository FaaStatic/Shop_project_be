package middleware

import (
	"encoding/base64"

	"github.com/gofiber/fiber/v3/middleware/encryptcookie"
)

// isValidEncryptKey reports whether key is a valid encryptcookie key: a
// base64-encoded string that decodes to 16, 24, or 32 bytes (AES-128/192/256).
// This mirrors encryptcookie's own validation so a misconfigured key falls back
// to a freshly generated one instead of panicking at startup.
func isValidEncryptKey(key string) bool {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return false
	}
	switch len(decoded) {
	case 16, 24, 32:
		return true
	default:
		return false
	}
}

// GetSecureCookiesMiddleware returns the encryptcookie middleware config. In
// production a configured key is used only if it is a valid base64-encoded
// 16/24/32-byte key; otherwise (and in every non-production environment) a
// fresh 32-byte key is generated so the app never panics on a bad key.
func GetSecureCookiesMiddleware(env string, encryptKey string) encryptcookie.Config {
	var key string
	if env == "production" && encryptKey != "" && isValidEncryptKey(encryptKey) {
		key = encryptKey
	} else {
		key = encryptcookie.GenerateKey(32)
	}

	return encryptcookie.Config{
		Key: key,
	}
}
