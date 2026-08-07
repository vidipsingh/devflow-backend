package service

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "strings"
    "time"

    "devflow-backend/internal/models"
    "devflow-backend/internal/repository"

    "github.com/google/generative-ai-go/genai"
    "go.mongodb.org/mongo-driver/v2/bson"
    "google.golang.org/api/option"
)

const geminiModel = "gemini-2.0-flash"

// geminiReviewResponse matches the JSON we ask Gemini to return
type geminiReviewResponse struct {
    Summary     string                `json:"summary"`
    Suggestions []models.AISuggestion `json:"suggestions"`
}

// ReviewPRAsync fetches file contents and calls Gemini. Safe to run in a goroutine
func ReviewPRAsync(ctx context.Context, pr *models.PullRequest, repoID bson.ObjectID) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		_ = markReview(ctx, pr.ID, &models.AIReview{
			Status: "skipped",
			ErrorMsg: "GEMINI_API_KEY not set",
		})
		return
	}

	// Mark as pending immediately
	_ = markReview(ctx, pr.ID, &models.AIReview{Status: "pending"})

	if len(pr.ChangedFiles) == 0 {
		now := time.Now()
		_ = markReview(ctx, pr.ID, &models.AIReview{
			Status: "skipped",
			Summary: "No changed files to review",
			ReviewedAt: &now,
		})
		return
	}

	// Fetch file contents for all changed files
	filesContents := buildFileContext(ctx, repoID, pr.HeadBranch, pr.ChangedFiles)

	review, err := callGemini(ctx, apiKey, pr, filesContents)
	if err != nil {
		log.Printf("[AI Review] PR #%d error: %v", pr.Number, err)
		errMsg := err.Error()
		_ = markReview(ctx, pr.ID, &models.AIReview{
			Status: "error",
			ErrorMsg: errMsg,
		})
		return
	}
	_ = markReview(ctx, pr.ID, review)
}

// TriggerReview is the public entrypoint for manual re-runs (called from handler)
func TriggerReview(ctx context.Context, ownerID bson.ObjectID, repoSlug string, number int) error {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil {
		return ErrRepoNotFound
	}
	pr, err := repository.FindPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return ErrPRNotFound
	}

	go func() {
		gctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		ReviewPRAsync(gctx, pr, repo.ID)
	}()
	return nil
}

// Helpers
func markReview(ctx context.Context, prID bson.ObjectID, review *models.AIReview) error {
	return repository.SetPRAIReview(ctx, prID, review)
}

func buildFileContext(ctx context.Context, repoID bson.ObjectID, branch string, filePaths []string) string {
	var sb strings.Builder
	for _, path := range filePaths {
		content, err := repository.GetFileContent(ctx, repoID, branch, path)
		if err != nil || content == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n\n=== %s ===\n%s", path, content))
	}
	return sb.String()
}

func callGemini(ctx context.Context, apiKey string, pr *models.PullRequest, filesContents string) (*models.AIReview, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(geminiModel)
	model.SetTemperature(0.2)

	prompt := buildPrompt(pr, filesContents)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	raw := extractText(resp)
	parsed, err := parseReview(raw)
	 if err != nil {
        return nil, fmt.Errorf("parse response: %w (raw: %.200s)", err, raw)
    }

	now := time.Now()
    return &models.AIReview{
        Status:      "done",
        Summary:     parsed.Summary,
        Suggestions: parsed.Suggestions,
        Model:       geminiModel,
        ReviewedAt:  &now,
    }, nil
}

func buildPrompt(pr *models.PullRequest, fileContents string) string {
    return fmt.Sprintf(`You are a senior software engineer doing a code review.
Only flag issues that would MEANINGFULLY improve: code modularity, robustness, error handling, security, or performance.
Do NOT report formatting, style, naming, or cosmetic issues.
If the code is clean and well-written, return an empty suggestions array — do not invent issues.

Pull Request: "%s"
Description: %s
Changed files: %s

--- FILE CONTENTS ---%s

Respond ONLY with valid JSON (no markdown, no code fences) matching this exact schema:
{
  "summary": "1-2 sentence overall assessment of the code quality",
  "suggestions": [
    {
      "filePath": "path/to/file.go",
      "lineStart": 10,
      "lineEnd": 15,
      "severity": "info|warning|critical",
      "category": "modularity|robustness|security|performance|error-handling",
      "message": "Concise description of the issue",
      "suggestion": "Specific actionable fix"
    }
  ]
}`,
        pr.Title,
        pr.Body,
        strings.Join(pr.ChangedFiles, ", "),
        fileContents,
    )
}

func extractText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}
	cand := resp.Candidates[0]
	if cand.Content == nil {
		return ""
	}
	var sb strings.Builder
	for _, part := range cand.Content.Parts {
		if t, ok := part.(genai.Text); ok {
			sb.WriteString(string(t))
		}
	}
	return strings.TrimSpace(sb.String())
}

func parseReview(raw string) (*geminiReviewResponse, error) {
	// Strip markdown code fences if Gemini wraps with them anyway
    raw = strings.TrimPrefix(raw, "```json")
    raw = strings.TrimPrefix(raw, "```")
    raw = strings.TrimSuffix(raw, "```")
    raw = strings.TrimSpace(raw)

	var result geminiReviewResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	if result.Suggestions == nil {
		result.Suggestions = []models.AISuggestion{}
	}
	return &result, nil
}
