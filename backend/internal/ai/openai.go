package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// OpenAIConfig holds configuration for the OpenAI client
type OpenAIConfig struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// OpenAIClient implements the Client interface using OpenAI's Vision API
type OpenAIClient struct {
	config     OpenAIConfig
	httpClient *http.Client
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(config OpenAIConfig) *OpenAIClient {
	if config.Model == "" {
		config.Model = "gpt-4o"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 250
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.Timeout == 0 {
		config.Timeout = 12 * time.Second
	}

	return &OpenAIClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// OpenAI API structures
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type openAITextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIImageContent struct {
	Type     string       `json:"type"`
	ImageURL openAIImgURL `json:"image_url"`
}

type openAIImgURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Storytell implements Client.Storytell
func (c *OpenAIClient) Storytell(hand []CardWithThumb) (*StorytellerResponse, error) {
	prompt := `You are playing Dixit. You are the storyteller this round.

Objective:
Create a clue that makes SOME (not all, not none) players guess your card.
You are trying to engineer an interesting split vote (ideally 1–(players-2) correct guesses).

Clue rules:
- Do NOT literally describe the image or list visible objects ("a cat", "a tower", "a moon", etc.).
- Do NOT mention colors, composition, or camera framing ("top-left", "close up").
- Avoid unique proper nouns that directly identify the card. Indirect references are okay (myth, proverb, classic story vibe).
- Prefer: metaphor, emotion, theme, relationship, contradiction, atmosphere, or a vague cultural echo.
- Clue length: 2–10 words. No emojis. No quotes.

Strategy:
- If your card feels very distinctive, choose a clue that could plausibly match 1–2 other cards.
- If your hand is uniformly similar, choose a clue that distinguishes your chosen card subtly.

Task:
1) Choose exactly ONE card from your hand.
2) Provide exactly ONE clue.

Return ONLY valid JSON (no markdown, no extra keys):
{"selectedCard": <1-N>, "clue": "<text>"}

Self-check before responding:
- selectedCard is a valid number 1..N
- clue obeys the rules above`

	content := []interface{}{
		openAITextContent{Type: "text", Text: prompt},
	}

	for i, card := range hand {
		content = append(content,
			openAITextContent{Type: "text", Text: fmt.Sprintf("Card %d:", i+1)},
			openAIImageContent{
				Type: "image_url",
				ImageURL: openAIImgURL{
					URL:    card.DataURL,
					Detail: "low",
				},
			},
		)
	}

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanJSONResponse(respBody)

	var response StorytellerResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}
	if response.Clue == "" {
		return nil, fmt.Errorf("empty clue in response")
	}

	return &response, nil
}

// Submit implements Client.Submit
func (c *OpenAIClient) Submit(hand []CardWithThumb, clue string) (*SubmitResponse, error) {
	prompt := fmt.Sprintf(`You are playing Dixit. You are NOT the storyteller.

The storyteller's clue is: "%s"

Objective:
Submit ONE card from your hand that will attract votes by seeming like it could be the storyteller's card.

Important:
- You are trying to be a believable decoy, not the "best match" in a literal sense.
- Think: mood, symbolism, implied story, emotional tone, genre.

Strategy:
- If the clue is abstract: pick a card with strong atmosphere or symbolism.
- If the clue hints at a narrative: pick a card that suggests a similar story arc.
- Avoid being too perfect (obvious storyteller) or too random (no votes).

Return ONLY valid JSON (no markdown, no extra keys):
{"selectedCard": <1-N>}

Self-check:
- selectedCard is 1..N`, clue)

	content := []interface{}{
		openAITextContent{Type: "text", Text: prompt},
	}

	for i, card := range hand {
		content = append(content,
			openAITextContent{Type: "text", Text: fmt.Sprintf("Card %d:", i+1)},
			openAIImageContent{
				Type: "image_url",
				ImageURL: openAIImgURL{
					URL:    card.DataURL,
					Detail: "low",
				},
			},
		)
	}

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanJSONResponse(respBody)

	var response SubmitResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(hand) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(hand))
	}

	return &response, nil
}

// Vote implements Client.Vote
func (c *OpenAIClient) Vote(submissions []CardWithThumb, clue string, ownIndex int) (*VoteResponse, error) {
	prompt := fmt.Sprintf(`You are playing Dixit. You are voting for which card is the STORYTELLER'S.

The storyteller's clue is: "%s"

Look at all the submitted cards and vote for the one you think is the STORYTELLER'S card (not your own).

IMPORTANT: Card %d is YOUR card - you cannot vote for it! Choose a different card.

How to decide:
- Interpret the clue as the storyteller intended: metaphor, mood, theme, indirect reference.
- Prefer the card that feels like it inspired the clue rather than one that merely "fits".
- If multiple cards fit, choose the one with the most storyteller-like intent (creative, central, evocative).

Return ONLY valid JSON (no markdown, no extra keys):
{"selectedCard": <1-N>}

Self-check:
- selectedCard is 1..N and not <OWN>
Remember: You CANNOT vote for card %d (your own card).`, clue, ownIndex, ownIndex)

	content := []interface{}{
		openAITextContent{Type: "text", Text: prompt},
	}

	for i, card := range submissions {
		label := fmt.Sprintf("Card %d:", i+1)
		if i+1 == ownIndex {
			label = fmt.Sprintf("Card %d (YOUR CARD - cannot vote):", i+1)
		}
		content = append(content,
			openAITextContent{Type: "text", Text: label},
			openAIImageContent{
				Type: "image_url",
				ImageURL: openAIImgURL{
					URL:    card.DataURL,
					Detail: "low",
				},
			},
		)
	}

	respBody, err := c.sendRequest(content)
	if err != nil {
		return nil, err
	}

	// Clean markdown code blocks from response
	cleanedBody := cleanJSONResponse(respBody)

	var response VoteResponse
	if err := json.Unmarshal([]byte(cleanedBody), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, respBody)
	}

	// Validate response
	if response.SelectedCard < 1 || response.SelectedCard > len(submissions) {
		return nil, fmt.Errorf("invalid card selection: %d (must be 1-%d)", response.SelectedCard, len(submissions))
	}
	if response.SelectedCard == ownIndex {
		return nil, fmt.Errorf("AI voted for its own card (%d)", ownIndex)
	}

	return &response, nil
}

// sendRequest sends a request to the OpenAI API and returns the response text
func (c *OpenAIClient) sendRequest(content []interface{}) (string, error) {
	reqBody := openAIRequest{
		Model: c.config.Model,
		Messages: []openAIMessage{
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return "", fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in OpenAI response")
	}

	return openAIResp.Choices[0].Message.Content, nil
}

// cleanJSONResponse strips markdown code blocks and extra whitespace from AI responses
func cleanJSONResponse(s string) string {
	// Remove markdown code blocks (```json ... ``` or ``` ... ```)
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	if matches := re.FindStringSubmatch(s); len(matches) > 1 {
		s = matches[1]
	}

	// Trim whitespace
	s = strings.TrimSpace(s)

	return s
}
