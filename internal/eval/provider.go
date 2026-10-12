package eval

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAI struct{ Client openai.Client }

func NewOpenAI() (*OpenAI, error) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set; provide it in the process environment")
	}
	return &OpenAI{openai.NewClient(option.WithMaxRetries(0))}, nil
}
func (p *OpenAI) Complete(ctx context.Context, r Request) (Response, error) {
	params := responses.ResponseNewParams{
		Model:        shared.ResponsesModel(r.Model.Name),
		Instructions: openai.String(r.Instructions),
		Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(r.Input)},
		Store:        openai.Bool(false), MaxOutputTokens: openai.Int(r.Model.MaxOutputTokens),
	}
	if r.Model.Effort != "" {
		params.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(r.Model.Effort)}
	}
	if r.Schema != nil {
		params.Text = responses.ResponseTextConfigParam{Format: responses.ResponseFormatTextConfigUnionParam{OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{Name: "writing_judgment", Schema: r.Schema, Strict: openai.Bool(true)}}}
	}
	result, err := p.Client.Responses.New(ctx, params)
	if err != nil {
		return Response{}, err
	}
	out := Response{ID: result.ID, Model: string(result.Model), Status: string(result.Status), Text: result.OutputText(), Usage: Usage{Input: result.Usage.InputTokens, CachedInput: result.Usage.InputTokensDetails.CachedTokens, Output: result.Usage.OutputTokens}}
	for _, item := range result.Output {
		if item.Type == "message" {
			for _, content := range item.Content {
				if content.Type == "refusal" {
					out.Status = "refused"
				}
			}
		}
	}
	if out.Status != "completed" {
		return out, fmt.Errorf("response status: %s", out.Status)
	}
	if strings.TrimSpace(out.Text) == "" {
		return out, fmt.Errorf("empty response")
	}
	return out, nil
}
func retryable(err error) bool {
	var api *openai.Error
	if errors.As(err, &api) {
		return api.StatusCode == 429 || api.StatusCode == 408 || api.StatusCode >= 500
	}
	var network net.Error
	return errors.As(err, &network) || errors.Is(err, context.DeadlineExceeded)
}

// Avoid persisting provider error bodies, which may include request headers or credentials.
func safeError(err error) string {
	var api *openai.Error
	if errors.As(err, &api) {
		return fmt.Sprintf("OpenAI HTTP %d (%s)", api.StatusCode, api.Code)
	}
	if errors.Is(err, context.Canceled) {
		return "request canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "request deadline exceeded"
	}
	var network net.Error
	if errors.As(err, &network) {
		return "network error"
	}
	return err.Error()
}
