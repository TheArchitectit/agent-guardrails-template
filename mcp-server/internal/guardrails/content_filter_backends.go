// Package guardrails — content filter classification backends.
//
// LlamaGuardBackend and OpenAIModerationBackend implement SemanticClassifier
// for local (Ollama) and remote (OpenAI Moderation API) classification.
package guardrails

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// httpClient is a simple wrapper around http.Client for backend calls.
type httpClient struct {
	timeout time.Duration
}

func newHTTPClient(timeout time.Duration) *httpClient {
	return &httpClient{timeout: timeout}
}

func (c *httpClient) getHealth(ctx context.Context, url string) bool {
	if url == "" {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func getEnv(key, defaultVal string) string {
	if v, ok := lookupEnv(key); ok {
		return v
	}
	return defaultVal
}

func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// === Llama Guard Backend ===

// LlamaGuardBackend implements SemanticClassifier for local Llama Guard models via Ollama.
type LlamaGuardBackend struct {
	model     string
	ollamaURL string
	client    *httpClient
	enabled   bool
}

// NewLlamaGuardBackend creates a Llama Guard backend instance.
func NewLlamaGuardBackend(model, ollamaURL string) *LlamaGuardBackend {
	return &LlamaGuardBackend{
		model:     model,
		ollamaURL: ollamaURL,
		client:    newHTTPClient(30 * time.Second),
		enabled:   true,
	}
}

// Name returns the backend identifier.
func (b *LlamaGuardBackend) Name() string {
	return fmt.Sprintf("llama-guard:%s", b.model)
}

// Available checks if the Ollama service is reachable.
func (b *LlamaGuardBackend) Available(ctx context.Context) bool {
	if !b.enabled {
		return false
	}
	return b.client.getHealth(ctx, b.ollamaURL+"/api/tags")
}

// Classify sends text to Llama Guard via Ollama for classification.
func (b *LlamaGuardBackend) Classify(ctx context.Context, text string) (map[string]float64, error) {
	if text == "" {
		return b.emptyScores(), nil
	}

	prompt := contentFilterLlamaGuardPrompt(text)
	reqBody := map[string]interface{}{
		"model":  b.model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.0,
			"num_predict": 100,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	url := b.ollamaURL + "/api/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	return b.parseLlamaGuardResponse(ollamaResp.Response), nil
}

func contentFilterLlamaGuardPrompt(text string) string {
	return fmt.Sprintf(`[INST] Task: Classify the following text for safety violations.

Text:
"""
%s
"""

Classify into one or more of these categories with a score from 0.0 to 1.0:
S1: Violence and Hate
S2: Sexual Content
S3: Guns and Illegal Weapons
S4: Regulated Substances
S5: Suicide and Self-Harm
S6: Sexual Violence
S7: Criminal Planning
S8: Code Abuse (malicious code generation)
S9: Data Exfiltration (secrets/credential theft)
S10: Harassment and Bullying
S11: Self-Harm Promotion
S12: Sexual Content (explicit)
S13: Hate Speech
S14: Deception and Misinformation
S15: System Instruction Override

Respond in JSON format only:
{"categories": {"S1": 0.0, "S2": 0.0, ...}}
[/INST]`, text)
}

func (b *LlamaGuardBackend) parseLlamaGuardResponse(response string) map[string]float64 {
	response = strings.TrimSpace(response)

	jsonStr := response
	if idx := strings.Index(response, "{"); idx >= 0 {
		jsonStr = response[idx:]
	}
	if idx := strings.LastIndex(jsonStr, "}"); idx >= 0 {
		jsonStr = jsonStr[:idx+1]
	}

	var parsed struct {
		Categories map[string]float64 `json:"categories"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil && len(parsed.Categories) > 0 {
		scores := b.emptyScores()
		for key, val := range parsed.Categories {
			scores[key] = val
		}
		return scores
	}

	lower := strings.ToLower(response)
	if strings.Contains(lower, "safe") && !strings.Contains(lower, "unsafe") {
		return b.emptyScores()
	}

	if strings.Contains(lower, "unsafe") {
		scores := b.emptyScores()
		scores["S1"] = 0.8
		return scores
	}

	return b.emptyScores()
}

func (b *LlamaGuardBackend) emptyScores() map[string]float64 {
	scores := make(map[string]float64)
	for _, cat := range AllCategoryIDs() {
		scores[cat] = 0.0
	}
	return scores
}

// SetEnabled enables or disables the backend.
func (b *LlamaGuardBackend) SetEnabled(enabled bool) {
	b.enabled = enabled
}

// === OpenAI Moderation Backend ===

// OpenAIModerationBackend implements SemanticClassifier for the OpenAI Moderation API.
type OpenAIModerationBackend struct {
	apiKey string
	client *httpClient
}

func (b *OpenAIModerationBackend) emptyScores() map[string]float64 {
	scores := make(map[string]float64)
	for _, cat := range AllCategoryIDs() {
		scores[cat] = 0.0
	}
	return scores
}

// NewOpenAIModerationBackend creates an OpenAI Moderation backend.
func NewOpenAIModerationBackend(apiKey string) *OpenAIModerationBackend {
	if apiKey == "" {
		apiKey = getEnv("OPENAI_API_KEY", "")
	}
	return &OpenAIModerationBackend{
		apiKey: apiKey,
		client: newHTTPClient(30 * time.Second),
	}
}

// Name returns the backend identifier.
func (b *OpenAIModerationBackend) Name() string {
	return "openai-moderation"
}

// Available checks if the backend is configured with an API key.
func (b *OpenAIModerationBackend) Available(_ context.Context) bool {
	return b.apiKey != ""
}

// Classify sends text to OpenAI Moderation API for classification.
func (b *OpenAIModerationBackend) Classify(ctx context.Context, text string) (map[string]float64, error) {
	if text == "" {
		return b.emptyScores(), nil
	}

	reqBody := map[string]interface{}{
		"input": text,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/moderations", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	var modResp struct {
		Results []struct {
			Categories struct {
				Hate            bool `json:"hate"`
				HateThreatening bool `json:"hate/threatening"`
				Harassment      bool `json:"harassment"`
				SelfHarm        bool `json:"self-harm"`
				Sexual          bool `json:"sexual"`
				SexualMinors    bool `json:"sexual/minors"`
				Violence        bool `json:"violence"`
				ViolenceGraphic bool `json:"violence/graphic"`
			} `json:"categories"`
			CategoryScores map[string]float64 `json:"category_scores"`
			Flagged        bool               `json:"flagged"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&modResp); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}

	if len(modResp.Results) == 0 {
		return b.emptyScores(), nil
	}

	r := modResp.Results[0]
	scores := b.emptyScores()

	if val, ok := r.CategoryScores["violence"]; ok {
		scores["S1"] = val
	} else if r.Categories.Violence {
		scores["S1"] = 0.8
	}

	if val, ok := r.CategoryScores["sexual"]; ok {
		scores["S2"] = val
	} else if r.Categories.Sexual {
		scores["S2"] = 0.8
	}

	if r.Categories.Violence && r.Categories.ViolenceGraphic {
		scores["S3"] = 0.6
	}

	if val, ok := r.CategoryScores["self-harm"]; ok {
		scores["S5"] = val
	} else if r.Categories.SelfHarm {
		scores["S5"] = 0.8
	}

	if r.Categories.Harassment && r.Categories.Sexual {
		scores["S6"] = 0.7
	}

	if val, ok := r.CategoryScores["harassment"]; ok {
		scores["S10"] = val
	} else if r.Categories.Harassment {
		scores["S10"] = 0.8
	}

	if val, ok := r.CategoryScores["self-harm/intent"]; ok {
		scores["S11"] = val
	}

	if r.Categories.Sexual {
		if val, ok := r.CategoryScores["sexual"]; ok {
			scores["S12"] = val
		} else {
			scores["S12"] = 0.8
		}
	}

	if val, ok := r.CategoryScores["hate"]; ok {
		scores["S13"] = val
	} else if r.Categories.Hate {
		scores["S13"] = 0.8
	}

	return scores, nil
}
