package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// coreToolsExtended returns part of the core guardrail tool set. Tool definitions are
// split across registry files to respect the project's 500-line file limit.
func coreToolsExtended() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "guardrail_check_uncertainty",
			Description: "Force self-reflection when confidence in next step is low",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"current_plan": map[string]any{
						"type":        "string",
						"description": "Description of the current plan",
					},
					"uncertainty_reason": map[string]any{
						"type":        "string",
						"description": "Reason for uncertainty",
					},
				},
				Required: []string{"current_plan", "uncertainty_reason"},
			},
		},
		{
			Name:        "guardrail_check_halt_conditions",
			Description: "Evaluate if current state requires manual human escalation",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"status": map[string]any{
						"type":        "string",
						"description": "Current system/task status",
					},
				},
			},
		},
		{
			Name:        "guardrail_record_halt",
			Description: "Record a system-forced halt event",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"reason": map[string]any{
						"type":        "string",
						"description": "Reason for the halt",
					},
				},
				Required: []string{"reason"},
			},
		},
		{
			Name:        "guardrail_acknowledge_halt",
			Description: "Acknowledged a previously recorded halt to resume operation",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"halt_id": map[string]any{
						"type":        "string",
						"description": "ID of the halt being acknowledged",
					},
				},
				Required: []string{"halt_id"},
			},
		},
		{
			Name:        "guardrail_validate_production_first",
			Description: "Ensure production changes are prioritized or isolated correctly",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path being modified",
					},
				},
			},
		},
		{
			Name:        "guardrail_detect_feature_creep",
			Description: "Analyze if changes exceed original task scope",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"task_id": map[string]any{
						"type":        "string",
						"description": "Original task identifier",
					},
					"current_changes": map[string]any{
						"type":        "string",
						"description": "Diff or summary of changes so far",
					},
				},
				Required: []string{"task_id", "current_changes"},
			},
		},
		{
			Name:        "guardrail_verify_fixes_intact",
			Description: "Ensure recent bugfixes haven't been regressed by new edits",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"bug_id": map[string]any{
						"type":        "string",
						"description": "Known bug ID or description",
					},
					"file_path": map[string]any{
						"type":        "string",
						"description": "File to check",
					},
				},
				Required: []string{"bug_id", "file_path"},
			},
		},
		{
			Name:        "guardrail_team_init",
			Description: "Initialize a new project team with roles and rules",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Name of the project",
					},
					"teams": map[string]any{
						"type":        "array",
						"items":       map[string]any{"type": "string"},
						"description": "List of team names to initialize",
					},
				},
				Required: []string{"project_name", "teams"},
			},
		},
		{
			Name:        "guardrail_team_list",
			Description: "List all active teams and their configurations",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Filter by project name",
					},
				},
			},
		},
		{
			Name:        "guardrail_team_config_get",
			Description: "Get detailed configuration for a specific team",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Name of the project",
					},
					"team_name": map[string]any{
						"type":        "string",
						"description": "Name of the team",
					},
				},
				Required: []string{"project_name", "team_name"},
			},
		},
		{
			Name:        "guardrail_team_config_update",
			Description: "Update rules or roles for an existing team",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Name of the project",
					},
					"team_name": map[string]any{
						"type":        "string",
						"description": "Name of the team",
					},
					"config": map[string]any{
						"type":        "object",
						"description": "New configuration data",
					},
				},
				Required: []string{"project_name", "team_name", "config"},
			},
		},
		{
			Name:        "guardrail_advisor_list",
			Description: "List all available AI advisors and their specialties",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
			},
		},
		{
			Name:        "guardrail_advisor_query",
			Description: "Ask a specialist AI advisor for guidance on a specific topic",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"advisor_name": map[string]any{
						"type":        "string",
						"description": "Name of the specialist advisor",
					},
					"query": map[string]any{
						"type":        "string",
						"description": "Your question or request",
					},
				},
				Required: []string{"advisor_name", "query"},
			},
		},
		{
			Name:        "guardrail_team_assign",
			Description: "Assign a specific team member (AI advisor) to a project",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Project name",
					},
					"advisor_name": map[string]any{
						"type":        "string",
						"description": "Advisor to assign",
					},
					"role": map[string]any{
						"type":        "string",
						"description": "Specific project role",
					},
				},
				Required: []string{"project_name", "advisor_name"},
			},
		},
		{
			Name:        "guardrail_team_remove",
			Description: "Remove a team or advisor assignment from a project",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Project name",
					},
					"team_id": map[string]any{
						"type":        "number",
						"description": "Team ID to delete (1-12)",
					},
					"confirmed": map[string]any{
						"type":        "boolean",
						"description": "Set to true to confirm deletion. First call without this to see confirmation prompt.",
					},
				},
			},
		},
		{
			Name:        "guardrail_project_delete",
			Description: "Delete an entire project and all its teams. Requires confirmation.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Name of the project to delete",
					},
					"confirmed": map[string]any{
						"type":        "boolean",
						"description": "Set to true to confirm deletion. First call without this to see confirmation prompt.",
					},
				},
			},
		},
		{
			Name:        "guardrail_team_health",
			Description: "Check team_manager.py health status - validates Python backend and file system access",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"project_name": map[string]any{
						"type":        "string",
						"description": "Optional: Project name for config directory check",
					},
				},
			},
		},
		{
			Name:        "guardrail_install_skills",
			Description: "Install or clone guardrails skill configs. Use 'skill' for per-skill install/clone, 'platforms' for full platform install, or 'path' for single-file clone.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"target_path": map[string]any{
						"type":        "string",
						"description": "Target project directory path (default: current directory)",
					},
					"platforms": map[string]any{
						"type":        "string",
						"description": "Comma-separated list of platforms: claude, cursor, opencode, windsurf, copilot (default: all). Use with action=install.",
					},
					"skill": map[string]any{
						"type":        "string",
						"description": "Install a single skill by name (e.g. 'guardrails-enforcer', 'commit-validator', 'four-laws'). Use action=install. Run list_skills=true to see all.",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Clone a single file by repo path (e.g. '.claude/skills/guardrails-enforcer.json'). Downloads from GitHub raw. Use with action=clone.",
					},
					"action": map[string]any{
						"type":        "string",
						"description": "Action to perform: 'install' (default), 'clone' (download from GitHub), 'list' (list skills/platforms)",
						"enum":        []string{"install", "clone", "list"},
					},
					"list_skills": map[string]any{
						"type":        "boolean",
						"description": "List all available skills and exit",
					},
					"list_platforms": map[string]any{
						"type":        "boolean",
						"description": "List all available platforms and exit",
					},
					"mode": map[string]any{
						"type":        "string",
						"description": "Installation mode: 'copy' or 'symlink' (default: copy). Applies to action=install.",
						"enum":        []string{"copy", "symlink"},
					},
					"dry_run": map[string]any{
						"type":        "boolean",
						"description": "Preview what would be done without making changes (default: false)",
					},
				},
			},
		},
	}
}
