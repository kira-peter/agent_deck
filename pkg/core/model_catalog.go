package core

// ModelFamily is a group of models from one vendor.
type ModelFamily struct {
	Name   string
	Models []string
}

// ModelCatalog lists the models offered by the relay, grouped by vendor.
// Within a vendor, newer series/versions come first.
var ModelCatalog = []ModelFamily{
	{"Claude", []string{
		"claude-opus-5.5",
		"claude-opus-5",
		"claude-opus-4.8",
		"claude-opus-4.7",
		"claude-opus-4.6",
		"claude-sonnet-5.5",
		"claude-sonnet-5",
		"claude-sonnet-4.6",
		"claude-haiku-5.5",
		"claude-haiku-4-5-20251001",
	}},
	{"GPT", []string{
		"gpt-6.1-sol",
		"gpt-6.1-terra",
		"gpt-6-sol",
		"gpt-6-luna",
		"gpt-6-astra",
		"gpt-5.6-sol",
		"gpt-5.6-terra",
		"gpt-5.6-luna",
		"codex-auto-review",
	}},
	{"Gemini", []string{
		"gemini-4-argon",
		"gemini-3.8-flash",
		"gemini-3.7-flash",
		"gemini-3.6-flash",
		"gemini-3.5-flash-lite",
		"gemini-3.1-pro-preview",
	}},
	{"DeepSeek", []string{"deepseek-v4-pro", "deepseek-v4-flash", "deepseek-flash"}},
	{"Doubao", []string{"doubao-seed-2.1-pro", "doubao-seed-2.1-turbo", "doubao-seed-2.1-lite"}},
	{"GLM", []string{"glm-5.3", "glm-5.3-flashx"}},
	{"Kimi", []string{"kimi-k3.1-pro", "kimi-k3"}},
	{"MiniMax", []string{"minimax-m3.1-flash", "minimax-m3"}},
	{"Grok", []string{"grok-4.7", "grok-4.6", "grok-4.5"}},
	{"Other", []string{"hy4-preview"}},
}

// FindModel locates a model id in the catalog.
func FindModel(id string) (family, index int, ok bool) {
	for f, fam := range ModelCatalog {
		for i, m := range fam.Models {
			if m == id {
				return f, i, true
			}
		}
	}
	return 0, 0, false
}
