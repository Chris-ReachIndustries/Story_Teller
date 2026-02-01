package images

import "strings"

// PromptBuilder constructs high-quality, theme-aware prompts for Stable Diffusion
type PromptBuilder struct {
	qualityKeywords []string
	dixitKeywords   []string
}

// NewPromptBuilder creates a new PromptBuilder instance
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		qualityKeywords: []string{
			"masterpiece",
			"best quality",
			"highly detailed",
			"sharp focus",
			"professional illustration",
			"coherent composition",
			"clear focal point",
		},
		dixitKeywords: []string{
			"evocative imagery",
			"dreamlike quality",
			"symbolic visual elements",
			"emotional depth",
		},
	}
}

// Build creates a complete prompt with theme context and quality keywords
// Structure: [theme style] + [card subject] + [dixit qualities] + [quality boost]
func (pb *PromptBuilder) Build(cardPrompt, theme, themeStyle string) string {
	var parts []string

	// Add theme style first for visual coherence
	if themeStyle != "" {
		parts = append(parts, themeStyle)
	}

	// Add the individual card prompt
	if cardPrompt != "" {
		parts = append(parts, cardPrompt)
	}

	// Add Dixit-specific qualities for interpretive imagery
	parts = append(parts, strings.Join(pb.dixitKeywords, ", "))

	// Add quality boosters
	parts = append(parts, strings.Join(pb.qualityKeywords, ", "))

	return strings.Join(parts, ", ")
}

// BuildNegativePrompt creates an enhanced negative prompt with anatomy fixes
func (pb *PromptBuilder) BuildNegativePrompt() string {
	anatomyFixes := []string{
		"bad anatomy",
		"bad proportions",
		"deformed",
		"disconnected limbs",
		"disfigured",
		"extra arms",
		"extra limbs",
		"extra hands",
		"fused fingers",
		"gross proportions",
		"malformed limbs",
		"mutated",
		"mutated hands",
		"missing fingers",
		"poorly drawn hands",
		"poorly drawn face",
		"long neck",
		"extra fingers",
		"missing arms",
	}

	qualityFixes := []string{
		"worst quality",
		"low quality",
		"normal quality",
		"lowres",
		"blurry",
		"text",
		"watermark",
		"logo",
		"signature",
		"username",
		"error",
		"jpeg artifacts",
		"cropped",
		"out of frame",
		"duplicate",
		"ugly",
		"disgusting",
		"nsfw",
	}

	allNegatives := append(anatomyFixes, qualityFixes...)
	return strings.Join(allNegatives, ", ")
}
