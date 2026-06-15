package budget

// ModelPricing holds per-million-token costs in cents.
type ModelPricing struct {
	InputPer1M  int64 // cost in cents per 1M input tokens
	OutputPer1M int64 // cost in cents per 1M output tokens
}

// DefaultPricing is the built-in pricing table.
// Values in cents per 1M tokens.
var DefaultPricing = map[string]ModelPricing{
	"claude-3-5-sonnet":  {InputPer1M: 300, OutputPer1M: 1500},
	"claude-3-opus":      {InputPer1M: 1500, OutputPer1M: 7500},
	"claude-3-haiku":     {InputPer1M: 25, OutputPer1M: 125},
	"gpt-4o":             {InputPer1M: 250, OutputPer1M: 1000},
	"gpt-4-turbo":        {InputPer1M: 1000, OutputPer1M: 3000},
	"gpt-3.5-turbo":      {InputPer1M: 50, OutputPer1M: 150},
	"local-llama":        {InputPer1M: 0, OutputPer1M: 0},
}

// EstimateCost calculates the cost in cents for a given model and token counts.
func EstimateCost(modelName string, inputTokens, outputTokens int64) int64 {
	pricing, ok := DefaultPricing[modelName]
	if !ok {
		// Unknown model — estimate zero cost rather than fail
		return 0
	}
	// (tokens / 1,000,000) * price_per_1M, using integer math
	inputCost := (inputTokens * pricing.InputPer1M) / 1_000_000
	outputCost := (outputTokens * pricing.OutputPer1M) / 1_000_000
	return inputCost + outputCost
}
