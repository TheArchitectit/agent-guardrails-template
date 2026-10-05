package notifications

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The dial guard is the delivery-time half of webhook SSRF defence:
// configuration-time validation can be outrun by a DNS answer that changes
// between configure and deliver, so the ADDRESS actually dialled is checked.
func TestPublicWebhookIPRejectsNonPublicDestinations(t *testing.T) {
	rejected := map[string]string{
		"loopback v4":        "127.0.0.1",
		"loopback v6":        "::1",
		"private 10/8":       "10.0.0.5",
		"private 192.168/16": "192.168.1.10",
		"private 172.16/12":  "172.16.9.9",
		"unspecified v4":     "0.0.0.0",
		"unspecified v6":     "::",
		"link-local v4":      "169.254.169.254",
		"link-local v6":      "fe80::1",
		"unique-local v6":    "fd00::1",
	}
	for name, raw := range rejected {
		ip := net.ParseIP(raw)
		if ip == nil {
			t.Fatalf("%s: fixture %q did not parse", name, raw)
		}
		if err := publicWebhookIP(ip); err == nil {
			t.Errorf("%s (%s): must be refused, was allowed", name, raw)
		}
	}

	for _, raw := range []string{"93.184.216.34", "2606:4700:4700::1111"} {
		ip := net.ParseIP(raw)
		if ip == nil {
			t.Fatalf("public fixture %q did not parse", raw)
		}
		if err := publicWebhookIP(ip); err != nil {
			t.Errorf("public address %s must be allowed, got %v", raw, err)
		}
	}

	if err := publicWebhookIP(nil); err == nil {
		t.Error("a nil address must be refused, not dialled")
	}
}

// A redirect would move the request to a target configuration-time validation
// never saw, so the delivery client must refuse to follow one.
func TestWebhookClientRefusesRedirects(t *testing.T) {
	var reached string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, backend.URL+"/moved", http.StatusFound)
	}))
	defer redirector.Close()

	client := newWebhookClient()
	req, err := http.NewRequest(http.MethodPost, redirector.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if _, err := client.Do(req); err == nil {
		t.Fatal("a redirect response must be refused, not followed")
	}
	if reached == "/moved" {
		t.Fatal("the redirect was followed; the second hop was never validated")
	}
}

// The client's dial guard must reject a loopback destination even when a
// caller reaches the transport directly — this is the rebinding seam.
func TestWebhookClientDialRefusesLoopback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newWebhookClient()
	req, err := http.NewRequest(http.MethodPost, srv.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if _, err := client.Do(req); err == nil {
		t.Fatal("a loopback destination must be refused at dial time")
	}
}
