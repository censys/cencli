package chartgen

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"github.com/censys/cencli/internal/app/chartgen/prompts"
	"github.com/censys/cencli/internal/pkg/cenclierrors"
)

const (
	// DefaultModel is the Gemini model used for image generation.
	DefaultModel = "gemini-3-pro-image-preview"
	// DefaultNumImages is the default number of images to generate.
	DefaultNumImages = 1
)

//go:generate mockgen -destination=../../../gen/app/chartgen/mocks/chartgenservice_mock.go -package=mocks -mock_names Service=MockChartgenService . Service

// Service generates charts from aggregation data using Gemini AI.
type Service interface {
	// GenerateChart generates chart images from aggregation data.
	GenerateChart(ctx context.Context, params Params) (Result, cenclierrors.CencliError)
}

type chartgenService struct {
	apiKey string
}

// New creates a new chartgen service with the given Gemini API key.
func New(apiKey string) Service {
	return &chartgenService{apiKey: apiKey}
}

func (s *chartgenService) GenerateChart(ctx context.Context, params Params) (Result, cenclierrors.CencliError) {
	// Create Gemini client
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  s.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return Result{}, cenclierrors.NewCencliError(fmt.Errorf("failed to create Gemini client: %w", err))
	}

	// Build prompt
	promptBuilder := prompts.
		New(params.Buckets, params.TotalCount, params.OtherCount).
		WithQuery(params.Query).
		WithField(params.Field)

	if params.ChartType != "" {
		promptBuilder = promptBuilder.WithChartType(params.ChartType)
	}

	prompt := promptBuilder.Build()

	// Determine number of images
	numImages := params.NumImages
	if numImages <= 0 {
		numImages = DefaultNumImages
	}

	// Generate images
	images := make([][]byte, 0, numImages)
	for i := 0; i < numImages; i++ {
		result, err := client.Models.GenerateContent(
			ctx,
			DefaultModel,
			genai.Text(prompt),
			&genai.GenerateContentConfig{},
		)
		if err != nil {
			return Result{}, cenclierrors.NewCencliError(fmt.Errorf("failed to generate image %d: %w", i+1, err))
		}

		// Extract image from response
		if len(result.Candidates) == 0 || result.Candidates[0].Content == nil {
			return Result{}, cenclierrors.NewCencliError(fmt.Errorf("no content in response for image %d", i+1))
		}

		for _, part := range result.Candidates[0].Content.Parts {
			if part.InlineData != nil && len(part.InlineData.Data) > 0 {
				images = append(images, part.InlineData.Data)
				break
			}
		}
	}

	if len(images) == 0 {
		return Result{}, cenclierrors.NewCencliError(fmt.Errorf("no images generated"))
	}

	return Result{
		Images: images,
		Prompt: prompt,
	}, nil
}
