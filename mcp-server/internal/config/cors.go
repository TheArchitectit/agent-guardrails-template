package config

import "fmt"

// defaultDevCORSOrigins is the convenience origin list used outside production
// when CORS_ALLOWED_ORIGINS is unset or a bare wildcard (Spec 12 §3.3.2.2).
var defaultDevCORSOrigins = []string{"http://localhost:*", "https://localhost:*"}

// EffectiveCORSOrigins resolves the origin list the server will actually use.
// It is the single place the non-production default substitution happens, so
// the wiring layer cannot silently widen exposure (Spec 12 §3.3.2):
//
//   - production: an empty list or a wildcard entry is a hard error — a
//     startup failure, never a substitution. config.Validate already rejects
//     this at load time; this method makes the wiring layer fail closed too,
//     so a directly-constructed Server cannot substitute either.
//   - non-production: an empty list or a bare "*" is replaced by
//     defaultDevCORSOrigins and substituted=true, so the caller logs a warning
//     naming the effective list.
func (c *Config) EffectiveCORSOrigins() (origins []string, substituted bool, err error) {
	if len(c.CORSAllowedOrigins) == 0 {
		if c.ProductionMode {
			return nil, false, fmt.Errorf("CORS_ALLOWED_ORIGINS must not be empty in production")
		}
		return append([]string(nil), defaultDevCORSOrigins...), true, nil
	}

	wildcard := false
	for _, origin := range c.CORSAllowedOrigins {
		if origin == "*" {
			wildcard = true
			break
		}
	}
	if wildcard {
		if c.ProductionMode {
			return nil, false, fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain wildcard '*' in production; list explicit origins")
		}
		return append([]string(nil), defaultDevCORSOrigins...), true, nil
	}

	return append([]string(nil), c.CORSAllowedOrigins...), false, nil
}
