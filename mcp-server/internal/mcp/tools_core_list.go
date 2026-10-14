package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// coreTools returns part of the core guardrail tool set. Tool definitions are
// split across registry files to respect the project's 500-line file limit.
func coreTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "guardrail_init_session",
			Description: "Initialize a new session with security parameters and session ID",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"user_id": map[string]any{
						"type":        "string",
						"description": "Unique identifier for the user",
					},
					"environment": map[string]any{
						"type":        "string",
						"description": "Target environment (development, staging, production)",
					},
				},
				Required: []string{"user_id"},
			},
		},
		{
			Name:        "guardrail_validate_bash",
			Description: "Validate a bash command against security policies and prevention rules",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The bash command to validate",
					},
					"working_dir": map[string]any{
						"type":        "string",
						"description": "Current working directory",
					},
				},
				Required: []string{"command"},
			},
		},
		{
			Name:        "guardrail_validate_file_edit",
			Description: "Validate a file edit operation (search and replace) against safety rules",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Path to the file being edited",
					},
					"old_string": map[string]any{
						"type":        "string",
						"description": "Text to be replaced",
					},
					"new_string": map[string]any{
						"type":        "string",
						"description": "Replacement text",
					},
					"session_token": map[string]any{
						"type":        "string",
						"description": "Active session token (optional; confirms the file was read before editing)",
					},
					"session_id": map[string]any{
						"type":        "string",
						"description": "Alternate session identifier (optional; used when session_token is absent)",
					},
				},
				Required: []string{"file_path", "old_string", "new_string"},
			},
		},
		{
			Name:        "guardrail_validate_git_operation",
			Description: "Validate a git operation (commit, push, branch) against policy",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"operation": map[string]any{
						"type":        "string",
						"description": "Git command to validate (e.g., commit, push)",
					},
					"args": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Arguments to the git command",
					},
				},
				Required: []string{"operation"},
			},
		},
		{
			Name:        "guardrail_pre_work_check",
			Description: "Perform a mandatory pre-work safety check before starting a new task",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"task_description": map[string]any{
						"type":        "string",
						"description": "Brief description of the planned task",
					},
				},
				Required: []string{"task_description"},
			},
		},
		{
			Name:        "guardrail_get_context",
			Description: "Get the current active guardrail context and applicable rules",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Current working directory or file path",
					},
				},
			},
		},
		{
			Name:        "guardrail_validate_scope",
			Description: "Verify if a file path is within authorized project scope",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Path to the file to validate",
					},
					"authorized_scope": map[string]any{
						"type":        "string",
						"description": "Root directory of the authorized scope",
					},
				},
				Required: []string{"file_path"},
			},
		},
		{
			Name:        "guardrail_validate_commit",
			Description: "Validate proposed commit message and changed files",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"message": map[string]any{
						"type":        "string",
						"description": "Commit message to validate",
					},
				},
				Required: []string{"message"},
			},
		},
		{
			Name:        "guardrail_prevent_regression",
			Description: "Check if changes might reintroduce known bugs or violate strict patterns",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"file_paths": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "Files being modified",
					},
					"code_content": map[string]any{
						"type":        "string",
						"description": "Code content to match against known regression patterns",
					},
				},
			},
		},
		{
			Name:        "guardrail_check_test_prod_separation",
			Description: "Enforce strict separation between test code and production code",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Path to the file being checked",
					},
					"environment": map[string]any{
						"type":        "string",
						"description": "Environment being checked (test or prod)",
					},
				},
				Required: []string{"file_path"},
			},
		},
		{
			Name:        "guardrail_validate_push",
			Description: "Pre-push validation of current branch status and health",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"branch": map[string]any{
						"type":        "string",
						"description": "Branch to be pushed",
					},
					"is_force": map[string]any{
						"type":        "boolean",
						"description": "Whether the push is a force push",
					},
					"has_unpushed_commits": map[string]any{
						"type":        "boolean",
						"description": "Whether there are unpushed commits",
					},
				},
				Required: []string{"branch"},
			},
		},
		{
			Name:        "guardrail_record_file_read",
			Description: "Record that a file has been read by the agent (Four Laws enforcement)",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Path to the file that was read",
					},
					"session_token": map[string]any{
						"type":        "string",
						"description": "Active session token",
					},
				},
				Required: []string{"file_path", "session_token"},
			},
		},
		{
			Name:        "guardrail_record_attempt",
			Description: "Record a tool use attempt for tracking progress/failure rates",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"session_token": map[string]any{
						"type":        "string",
						"description": "Active session token",
					},
					"task_id": map[string]any{
						"type":        "string",
						"description": "Unique identifier for the current task",
					},
					"error_message": map[string]any{
						"type":        "string",
						"description": "Error message if failed (defaults to 'Unknown error')",
					},
					"error_category": map[string]any{
						"type":        "string",
						"description": "Error category (defaults to 'other')",
					},
				},
				Required: []string{"session_token"},
			},
		},
		{
			Name:        "guardrail_verify_file_read",
			Description: "Verify a file has been read in current context before editing",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "Path to the file to verify",
					},
					"session_token": map[string]any{
						"type":        "string",
						"description": "Active session token",
					},
				},
				Required: []string{"file_path", "session_token"},
			},
		},
		{
			Name:        "guardrail_validate_three_strikes",
			Description: "Check if current task has hit consecutive failure threshold",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"session_token": map[string]any{
						"type":        "string",
						"description": "Active session token",
					},
					"task_id": map[string]any{
						"type":        "string",
						"description": "Unique identifier for the current task",
					},
				},
				Required: []string{"session_token"},
			},
		},
		{
			Name:        "guardrail_validate_exact_replacement",
			Description: "Verify that strings for replacement exactly match target file content",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"file_path": map[string]any{
						"type":        "string",
						"description": "File path to check",
					},
					"session_token": map[string]any{
						"type":        "string",
						"description": "Active session token",
					},
					"original_content": map[string]any{
						"type":        "string",
						"description": "Original file content before the edit",
					},
					"modified_content": map[string]any{
						"type":        "string",
						"description": "Modified file content after the edit",
					},
					"replacement_type": map[string]any{
						"type":        "string",
						"description": "Type of replacement being validated",
					},
				},
				Required: []string{"file_path", "session_token"},
			},
		},
		{
			Name:        "guardrail_reset_attempts",
			Description: "Reset failure counters for a given task or tool",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"task_id": map[string]any{
						"type":        "string",
						"description": "Unique identifier for the current task",
					},
					"session_token": map[string]any{
						"type":        "string",
						"description": "Active session token",
					},
				},
				Required: []string{"session_token"},
			},
		},
	}
}
