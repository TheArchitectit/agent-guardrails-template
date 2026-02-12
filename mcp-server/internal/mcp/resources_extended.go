package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// Base path for documentation files (relative to project root)
const docsBasePath = "/mnt/ollama/git/agent-guardrails-template"

// readAgentGuardrailsResource reads the AGENT_GUARDRAILS.md documentation
func (s *MCPServer) readAgentGuardrailsResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	content, err := os.ReadFile(filepath.Join(docsBasePath, "docs", "AGENT_GUARDRAILS.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to read agent guardrails: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []interface{}{
			mcp.TextResourceContents{
				Uri:      uri,
				MimeType: "text/markdown",
				Text:     string(content),
			},
		},
	}, nil
}

// readWorkflowsResource lists all workflow documentation
func (s *MCPServer) readWorkflowsResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	workflowsPath := filepath.Join(docsBasePath, "docs", "workflows")

	// Read INDEX.md if it exists
	indexContent := ""
	indexPath := filepath.Join(workflowsPath, "INDEX.md")
	if data, err := os.ReadFile(indexPath); err == nil {
		indexContent = string(data)
	}

	// List all files in the workflows directory
	files, err := os.ReadDir(workflowsPath)
	if err != nil {
		// Return just the index if we can't read directory
		return &mcp.ReadResourceResult{
			Contents: []interface{}{
				mcp.TextResourceContents{
					Uri:      uri,
					MimeType: "text/markdown",
					Text:     indexContent,
				},
			},
		}, nil
	}

	// Build content with index and file listing
	var sb strings.Builder
	if indexContent != "" {
		sb.WriteString(indexContent)
		sb.WriteString("\n\n---\n\n")
	}
	sb.WriteString("## Available Workflow Files\n\n")

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}
		sb.WriteString(fmt.Sprintf("- `%s`\n", file.Name()))
	}

	return &mcp.ReadResourceResult{
		Contents: []interface{}{
			mcp.TextResourceContents{
				Uri:      uri,
				MimeType: "text/markdown",
				Text:     sb.String(),
			},
		},
	}, nil
}

// readStandardsResource lists all standards documentation
func (s *MCPServer) readStandardsResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	standardsPath := filepath.Join(docsBasePath, "docs", "standards")

	// Read INDEX.md if it exists
	indexContent := ""
	indexPath := filepath.Join(standardsPath, "INDEX.md")
	if data, err := os.ReadFile(indexPath); err == nil {
		indexContent = string(data)
	}

	// List all files in the standards directory
	files, err := os.ReadDir(standardsPath)
	if err != nil {
		// Return just the index if we can't read directory
		return &mcp.ReadResourceResult{
			Contents: []interface{}{
				mcp.TextResourceContents{
					Uri:      uri,
					MimeType: "text/markdown",
					Text:     indexContent,
				},
			},
		}, nil
	}

	// Build content with index and file listing
	var sb strings.Builder
	if indexContent != "" {
		sb.WriteString(indexContent)
		sb.WriteString("\n\n---\n\n")
	}
	sb.WriteString("## Available Standard Files\n\n")

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
			continue
		}
		sb.WriteString(fmt.Sprintf("- `%s`\n", file.Name()))
	}

	return &mcp.ReadResourceResult{
		Contents: []interface{}{
			mcp.TextResourceContents{
				Uri:      uri,
				MimeType: "text/markdown",
				Text:     sb.String(),
			},
		},
	}, nil
}

// readFourLawsResource reads the Four Laws of Agent Safety
func (s *MCPServer) readFourLawsResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	content, err := os.ReadFile(filepath.Join(docsBasePath, "skills", "shared-prompts", "four-laws.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to read four laws: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []interface{}{
			mcp.TextResourceContents{
				Uri:      uri,
				MimeType: "text/markdown",
				Text:     string(content),
			},
		},
	}, nil
}

// readHaltConditionsResource reads the halt conditions documentation
func (s *MCPServer) readHaltConditionsResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	content, err := os.ReadFile(filepath.Join(docsBasePath, "skills", "shared-prompts", "halt-conditions.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to read halt conditions: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []interface{}{
			mcp.TextResourceContents{
				Uri:      uri,
				MimeType: "text/markdown",
				Text:     string(content),
			},
		},
	}, nil
}

// readPreWorkChecklistResource reads the pre-work checklist
func (s *MCPServer) readPreWorkChecklistResource(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
	content, err := os.ReadFile(filepath.Join(docsBasePath, ".guardrails", "pre-work-check.md"))
	if err != nil {
		return nil, fmt.Errorf("failed to read pre-work checklist: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []interface{}{
			mcp.TextResourceContents{
				Uri:      uri,
				MimeType: "text/markdown",
				Text:     string(content),
			},
		},
	}, nil
}
