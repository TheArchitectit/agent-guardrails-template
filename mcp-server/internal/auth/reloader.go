package auth

import (
	"context"
	"log/slog"
	"time"
)

// RevocationReloadInterval is the period at which a running process re-reads
// its configured registry sources and swaps the validated snapshot in. It is
// deliberately well inside RevocationPropagationBound (60s): the published
// bound is the promise, this is the implementation that keeps it. A revoked
// record therefore denies a new request on every surface sharing this Registry
// within RevocationPropagationBound even though the process was never
// restarted.
const RevocationReloadInterval = 15 * time.Second

// Sources describes the configured out-of-band registry inputs. It is a plain
// value (no pointers into Config) so a reload can be driven from a package
// that must not import config.
type Sources struct {
	InlineJSON      string
	FilePath        string
	VerifierKey     string
	VerifierKeyFile string
}

// Configured reports whether any registry source is set.
func (s Sources) Configured() bool {
	return SourcesConfigured(s.InlineJSON, s.FilePath)
}

// Load builds a fresh, fully-validated registry from the sources. It returns
// (nil, nil) when no source is configured, matching LoadFromSourcesEx.
func (s Sources) Load() (*Registry, error) {
	return LoadFromSourcesEx(s.InlineJSON, s.FilePath, s.VerifierKey, s.VerifierKeyFile)
}

// ReloadOnce re-reads the sources and swaps the validated snapshot into this
// registry atomically. A failure (unreadable/malformed/empty source, bad key)
// leaves the current snapshot untouched: a failed reload never enlarges access
// and never falls back to the legacy surface. The verifier key is resolved the
// same way as at startup so a key supplied as a secret file keeps working.
func (r *Registry) ReloadOnce(s Sources) error {
	if r == nil {
		return nil
	}
	key, err := resolveVerifierKey(s.VerifierKey, s.VerifierKeyFile)
	if err != nil {
		return err
	}
	return r.ReloadFromSources(s.InlineJSON, s.FilePath, key)
}

// StartReloadLoop re-reads the sources every interval until ctx is cancelled,
// so a revocation visible in the source is enforced on the next tick. The
// first tick is scheduled one interval after start (the caller has already
// loaded the initial snapshot). A reload failure is reported through onErr and
// otherwise ignored: the previous validated snapshot stays in force, so a
// transient source outage narrows nothing and widens nothing.
func (r *Registry) StartReloadLoop(ctx context.Context, s Sources, interval time.Duration, onErr func(error)) {
	if r == nil || !s.Configured() {
		return
	}
	if interval <= 0 {
		interval = RevocationReloadInterval
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.ReloadOnce(s); err != nil {
					if onErr != nil {
						onErr(err)
					} else {
						slog.Warn("credential registry reload failed; previous validated snapshot remains in force", "error", err)
					}
				}
			}
		}
	}()
}
