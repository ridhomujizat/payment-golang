package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// PromptResult contains the response and metadata from a prompt
type PromptResult struct {
	Response     string
	TokenUsed    int
	ResponseTime int // in milliseconds
}

type AiClient struct {
	ctx          context.Context
	geminiClient *genai.Client
	geminiModel  string
}
type Config struct {
	GeminiAPIKey string
	GeminiModel  string
}

func NewAiClient(ctx context.Context, cfg *Config) *AiClient {
	aiClient := &AiClient{
		ctx: ctx,
	}

	if cfg.GeminiAPIKey != "" && cfg.GeminiModel != "" {
		client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GeminiAPIKey))
		if err != nil {
			log.Fatal("Gagal membuat klien Gemini:", err)
		}
		// Remove defer client.Close() - client will be used later

		aiClient.geminiClient = client
		aiClient.geminiModel = cfg.GeminiModel
	}

	return aiClient
}

func (a *AiClient) GeminiPrompt(prompt string) (*PromptResult, error) {
	if a.geminiClient == nil {
		return nil, fmt.Errorf("Gemini client is not initialized")
	}

	model := a.geminiClient.GenerativeModel(a.geminiModel)

	startTime := time.Now()
	resp, err := model.GenerateContent(a.ctx, genai.Text(prompt))
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("received empty or invalid response structure from Gemini")
	}
	part := resp.Candidates[0].Content.Parts[0]
	rawGeminiText := ""
	if textPart, ok := part.(genai.Text); ok {
		rawGeminiText = string(textPart)
	} else {
		return nil, fmt.Errorf("unexpected response part type")
	}

	// Extract token usage
	tokenUsed := 0
	if resp.UsageMetadata != nil {
		tokenUsed = int(resp.UsageMetadata.TotalTokenCount)
	}

	return &PromptResult{
		Response:     rawGeminiText,
		TokenUsed:    tokenUsed,
		ResponseTime: responseTime,
	}, nil
}

// GeminiPromptWithImage sends a prompt with a base64 encoded image to Gemini API
func (a *AiClient) GeminiPromptWithImage(prompt string, base64Image string) (*PromptResult, error) {
	if a.geminiClient == nil {
		return nil, fmt.Errorf("Gemini client is not initialized")
	}

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %w", err)
	}

	model := a.geminiClient.GenerativeModel(a.geminiModel)

	startTime := time.Now()
	// Create content with both text and image
	resp, err := model.GenerateContent(a.ctx,
		genai.Text(prompt),
		genai.ImageData("jpeg", imageData),
	)
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API with image: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("received empty or invalid response structure from Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	rawGeminiText := ""
	if textPart, ok := part.(genai.Text); ok {
		rawGeminiText = string(textPart)
	} else {
		return nil, fmt.Errorf("unexpected response part type")
	}

	// Extract token usage
	tokenUsed := 0
	if resp.UsageMetadata != nil {
		tokenUsed = int(resp.UsageMetadata.TotalTokenCount)
	}

	return &PromptResult{
		Response:     rawGeminiText,
		TokenUsed:    tokenUsed,
		ResponseTime: responseTime,
	}, nil
}

// GeminiPromptWithImageAndSchema sends a prompt with a base64 encoded image and JSON schema for structured output
func (a *AiClient) GeminiPromptWithImageAndSchema(prompt string, base64Image string, schema *genai.Schema) (*PromptResult, error) {
	if a.geminiClient == nil {
		return nil, fmt.Errorf("Gemini client is not initialized")
	}

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %w", err)
	}

	model := a.geminiClient.GenerativeModel(a.geminiModel)

	// Set response schema for structured output
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = schema

	startTime := time.Now()
	// Create content with both text and image
	resp, err := model.GenerateContent(a.ctx,
		genai.Text(prompt),
		genai.ImageData("jpeg", imageData),
	)
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API with image and schema: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("received empty or invalid response structure from Gemini")
	}

	part := resp.Candidates[0].Content.Parts[0]
	rawGeminiText := ""
	if textPart, ok := part.(genai.Text); ok {
		rawGeminiText = string(textPart)
	} else {
		return nil, fmt.Errorf("unexpected response part type")
	}

	// Extract token usage
	tokenUsed := 0
	if resp.UsageMetadata != nil {
		tokenUsed = int(resp.UsageMetadata.TotalTokenCount)
	}

	return &PromptResult{
		Response:     rawGeminiText,
		TokenUsed:    tokenUsed,
		ResponseTime: responseTime,
	}, nil
}

// GeminiPromptWithSchema sends a text prompt with JSON schema for structured output
func (a *AiClient) GeminiPromptWithSchema(prompt string, schema *genai.Schema) (*PromptResult, error) {
	if a.geminiClient == nil {
		return nil, fmt.Errorf("Gemini client is not initialized")
	}

	model := a.geminiClient.GenerativeModel(a.geminiModel)

	// Set response schema for structured output
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = schema

	startTime := time.Now()
	resp, err := model.GenerateContent(a.ctx, genai.Text(prompt))
	responseTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		return nil, fmt.Errorf("failed to call Gemini API with schema: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("received empty or invalid response structure from Gemini")
	}
	part := resp.Candidates[0].Content.Parts[0]
	rawGeminiText := ""
	if textPart, ok := part.(genai.Text); ok {
		rawGeminiText = string(textPart)
	} else {
		return nil, fmt.Errorf("unexpected response part type")
	}

	// Extract token usage
	tokenUsed := 0
	if resp.UsageMetadata != nil {
		tokenUsed = int(resp.UsageMetadata.TotalTokenCount)
	}

	return &PromptResult{
		Response:     rawGeminiText,
		TokenUsed:    tokenUsed,
		ResponseTime: responseTime,
	}, nil
}

// Close properly closes the Gemini client
func (a *AiClient) Close() error {
	if a.geminiClient != nil {
		return a.geminiClient.Close()
	}
	return nil
}
