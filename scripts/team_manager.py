#!/usr/bin/env python3
"""
Team Manager - Standardized Team Layout Manager

Manages team assignments, tracks phase progress, and validates
team composition against standardized enterprise layout.
"""

import argparse
import fcntl
import json
import os
import re
import sys
import tempfile
import traceback
from dataclasses import dataclass, asdict
from datetime import datetime
from pathlib import Path
from typing import List, Optional, Dict




class StructuredLogger:
    """JSON structured logging with correlation ID support."""

    def __init__(self, component: str, request_id: Optional[str] = None):
        self.component = component
        self.request_id = request_id

    def log(self, event_type: str, details: Dict, level: str = "info") -> None:
        """Log a structured JSON event."""
        log_entry = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "level": level.upper(),
            "component": self.component,
            "event": event_type,
            "details": details
        }
        if self.request_id:
            log_entry["request_id"] = self.request_id
        print(json.dumps(log_entry), file=sys.stderr)

    def info(self, event_type: str, details: Dict) -> None:
        """Log INFO level event."""
        self.log(event_type, details, "info")

    def warn(self, event_type: str, details: Dict) -> None:
        """Log WARN level event."""
        self.log(event_type, details, "warn")

    def error(self, event_type: str, details: Dict, exc_info: bool = False) -> None:
        """Log ERROR level event with optional exception info."""
        if exc_info:
            details = {**details, "stack_trace": traceback.format_exc()}
        self.log(event_type, details, "error")

    def debug(self, event_type: str, details: Dict) -> None:
        """Log DEBUG level event."""
        self.log(event_type, details, "debug")

def validate_project_name(name: str) -> None:
    """Validate project name to prevent command injection."""
    if not name:
        raise ValueError("project_name is required")
    if len(name) > 64:
        raise ValueError("project_name must be 64 characters or less")
    if not re.match(r'^[a-zA-Z0-9_-]+$', name):
        raise ValueError("project_name must contain only letters, numbers, hyphens, and underscores")


class PermissionDenied(Exception):
    """Raised when user lacks permission for an operation."""
    pass


class FileLockError(Exception):
    """Raised when file locking fails."""
    pass


class UserContext:
    """User session context with RBAC information."""

    # Role hierarchy: higher number = more permissions
    ROLE_LEVELS = {
        "viewer": 1,      # Can view only
        "team-lead": 2,   # Can modify their team's assignments
        "admin": 3        # Can modify everything
    }

    def __init__(self, user_id: str, role: str, team_id: Optional[int] = None):
        self.user_id = user_id
        self.role = role
        self.team_id = team_id

        if role not in self.ROLE_LEVELS:
            raise ValueError(f"Invalid role: {role}. Must be one of: {list(self.ROLE_LEVELS.keys())}")

    def has_permission(self, required_role: str) -> bool:
        """Check if user has at least the required role level."""
        return self.ROLE_LEVELS.get(self.role, 0) >= self.ROLE_LEVELS.get(required_role, 0)

    def can_modify_team(self, team_id: int) -> bool:
        """Check if user can modify a specific team."""
        if self.role == "admin":
            return True
        if self.role == "team-lead" and self.team_id == team_id:
            return True
        return False


class FileLock:
    """Cross-platform file locking using flock (Unix) or msvcrt (Windows)."""

    def __init__(self, lock_file_path: Path, timeout: float = 30.0):
        self.lock_file_path = lock_file_path
        self.timeout = timeout
        self.lock_file = None

    def __enter__(self):
        """Acquire exclusive lock."""
        self.lock_file_path.parent.mkdir(parents=True, exist_ok=True)
        self.lock_file = open(self.lock_file_path, 'w')

        try:
            # Use non-blocking flock first
            fcntl.flock(self.lock_file.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
        except (IOError, OSError):
            # Lock is held by another process
            import time
            start_time = time.time()
            while time.time() - start_time < self.timeout:
                try:
                    fcntl.flock(self.lock_file.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
                    break
                except (IOError, OSError):
                    time.sleep(0.1)
            else:
                self.lock_file.close()
                raise FileLockError(f"Could not acquire lock within {self.timeout}s")

        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Release lock and close file."""
        if self.lock_file:
            try:
                fcntl.flock(self.lock_file.fileno(), fcntl.LOCK_UN)
            except (IOError, OSError):
                pass
            finally:
                self.lock_file.close()


@dataclass
class Role:
    """Standard team role."""
    name: str
    responsibility: str
    deliverables: List[str]
    assigned_to: Optional[str] = None


@dataclass
class Team:
    """Standard team definition."""
    id: int
    name: str
    phase: str
    description: str
    roles: List[Role]
    exit_criteria: List[str]
    status: str = "not_started"  # not_started, active, completed, blocked
    started_at: Optional[str] = None
    completed_at: Optional[str] = None


class TeamManager:
    """Manages standardized team layout."""

    # Standard team definitions
    STANDARD_TEAMS = {
        # Phase 1: Strategy, Governance & Planning
        1: Team(
            id=1,
            name="Business & Product Strategy",
            phase="Phase 1: Strategy, Governance & Planning",
            description="The 'Why' - Business case and product strategy",
            roles=[
                Role("Business Relationship Manager", "Connects IT to C-suite",
                     ["Strategic alignment docs", "Executive briefings"]),
                Role("Lead Product Manager", "Owns long-term roadmap",
                     ["Product roadmap", "OKRs", "Feature prioritization"]),
                Role("Business Systems Analyst", "Translates business to technical",
                     ["Requirements specs", "User stories", "Acceptance criteria"]),
                Role("Financial Controller (FinOps)", "Approves budget and cloud spend",
                     ["Budget forecasts", "Cost projections", "Spend reports"]),
            ],
            exit_criteria=[
                "Business case approved",
                "Budget allocated",
                "Roadmap defined",
                "Success metrics established"
            ]
        ),
        2: Team(
            id=2,
            name="Enterprise Architecture",
            phase="Phase 1: Strategy, Governance & Planning",
            description="The 'Standards' - Technology vision and standards",
            roles=[
                Role("Chief Architect", "Sets 5-year tech vision",
                     ["Architecture vision", "Tech radar", "Strategic plans"]),
                Role("Domain Architect", "Specialized stack expertise",
                     ["Domain-specific patterns", "Best practices guides"]),
                Role("Solution Architect", "Maps projects to standards",
                     ["Solution designs", "Architecture decision records"]),
                Role("Standards Lead", "Manages Approved Tech List",
                     ["Technology standards", "Evaluation criteria", "Approved list"]),
            ],
            exit_criteria=[
                "Architecture approved",
                "Technology choices validated",
                "Standards compliance verified"
            ]
        ),
        3: Team(
            id=3,
            name="GRC (Governance, Risk, & Compliance)",
            phase="Phase 1: Strategy, Governance & Planning",
            description="Compliance and risk management",
            roles=[
                Role("Compliance Officer", "SOX/HIPAA/GDPR adherence",
                     ["Compliance checklists", "Audit reports"]),
                Role("Internal Auditor", "Pre-production mock audits",
                     ["Audit findings", "Remediation plans"]),
                Role("Privacy Engineer", "Data masking and PII",
                     ["Privacy impact assessments", "Data flow diagrams"]),
                Role("Policy Manager", "Maintains SOPs",
                     ["Standard operating procedures", "Policy updates"]),
            ],
            exit_criteria=[
                "Compliance review passed",
                "Risk assessment complete",
                "Privacy requirements met",
                "Policies acknowledged"
            ]
        ),
        # Phase 2: Platform & Foundation
        4: Team(
            id=4,
            name="Infrastructure & Cloud Ops",
            phase="Phase 2: Platform & Foundation",
            description="Cloud infrastructure and networking",
            roles=[
                Role("Cloud Architect", "VPC and network design",
                     ["Network diagrams", "Security groups", "Routing tables"]),
                Role("IaC Engineer", "Provisions the 'metal'",
                     ["Terraform modules", "Ansible playbooks", "Infrastructure code"]),
                Role("Network Security Engineer", "Firewalls, VPNs, Direct Connect",
                     ["Security rules", "Network policies", "Access controls"]),
                Role("Storage Engineer", "S3/SAN management",
                     ["Storage policies", "Backup strategies", "Archival rules"]),
            ],
            exit_criteria=[
                "Infrastructure provisioned",
                "Network connectivity verified",
                "Security rules applied",
                "Monitoring enabled"
            ]
        ),
        5: Team(
            id=5,
            name="Platform Engineering",
            phase="Phase 2: Platform & Foundation",
            description="The 'Internal Tools' - Developer experience platform",
            roles=[
                Role("Platform Product Manager", "Developer experience as product",
                     ["Platform roadmap", "DX metrics", "Adoption reports"]),
                Role("CI/CD Architect", "Golden pipelines",
                     ["Pipeline templates", "Build configs", "Deployment strategies"]),
                Role("Kubernetes Administrator", "Cluster management",
                     ["Cluster configs", "Resource quotas", "Ingress rules"]),
                Role("Developer Advocate", "Dev squad adoption",
                     ["Onboarding guides", "Training materials", "Feedback loops"]),
            ],
            exit_criteria=[
                "Platform services ready",
                "CI/CD pipelines functional",
                "Developer onboarding complete"
            ]
        ),
        6: Team(
            id=6,
            name="Data Governance & Analytics",
            phase="Phase 2: Platform & Foundation",
            description="Enterprise data management",
            roles=[
                Role("Data Architect", "Enterprise data model",
                     ["Data models", "Schema designs", "Lineage documentation"]),
                Role("DBA", "Production database performance",
                     ["Query optimization", "Index tuning", "Backup verification"]),
                Role("Data Privacy Officer", "Retention and deletion rules",
                     ["Data retention policies", "Deletion workflows"]),
                Role("ETL Developer", "Data flow management",
                     ["ETL pipelines", "Data quality checks", "Transformation logic"]),
            ],
            exit_criteria=[
                "Data models defined",
                "Pipelines operational",
                "Privacy controls implemented"
            ]
        ),
        # Phase 3: The Build Squads
        7: Team(
            id=7,
            name="Core Feature Squad",
            phase="Phase 3: The Build Squads",
            description="The 'Devs' - Feature implementation",
            roles=[
                Role("Technical Lead", "Final word on implementation",
                     ["Code reviews", "Architecture decisions", "Technical guidance"]),
                Role("Senior Backend Engineer", "Logic, APIs, microservices",
                     ["Backend services", "API endpoints", "Business logic"]),
                Role("Senior Frontend Engineer", "Design system, state management",
                     ["UI components", "Frontend architecture", "State logic"]),
                Role("Accessibility (A11y) Expert", "WCAG compliance",
                     ["A11y audits", "Remediation plans", "Testing reports"]),
                Role("Technical Writer", "Internal/external docs",
                     ["API docs", "User guides", "Runbooks"]),
            ],
            exit_criteria=[
                "Features implemented",
                "Code reviewed and approved",
                "Documentation complete",
                "A11y requirements met"
            ]
        ),
        8: Team(
            id=8,
            name="Middleware & Integration",
            phase="Phase 3: The Build Squads",
            description="APIs and system integrations",
            roles=[
                Role("API Product Manager", "API lifecycle and versioning",
                     ["API specs", "Versioning strategy", "Deprecation plans"]),
                Role("Integration Engineer", "SAP/Oracle/Mainframe connections",
                     ["Integration specs", "Data mappings", "Error handling"]),
                Role("Messaging Engineer", "Kafka/RabbitMQ management",
                     ["Topic design", "Message schemas", "Consumer groups"]),
                Role("IAM Specialist", "Okta/AD integration",
                     ["Auth flows", "Permission models", "Access policies"]),
            ],
            exit_criteria=[
                "APIs documented and tested",
                "Integrations verified",
                "Auth flows functional"
            ]
        ),
        # Phase 4: Validation & Hardening
        9: Team(
            id=9,
            name="Cybersecurity (AppSec)",
            phase="Phase 4: Validation & Hardening",
            description="Application security",
            roles=[
                Role("Security Architect", "Threat model review",
                     ["Threat models", "Security architecture", "Risk assessments"]),
                Role("Vulnerability Researcher", "SAST/DAST/SCA scanners",
                     ["Scan reports", "Vulnerability triage", "Fix verification"]),
                Role("Penetration Tester", "Manual security testing",
                     ["Pen test reports", "Exploit verification", "Remediation"]),
                Role("DevSecOps Engineer", "Security in CI/CD",
                     ["Security gates", "Pipeline integration", "Compliance checks"]),
            ],
            exit_criteria=[
                "Security review passed",
                "Vulnerabilities remediated or accepted",
                "Pen testing complete",
                "Security gates passing"
            ]
        ),
        10: Team(
            id=10,
            name="Quality Engineering (SDET)",
            phase="Phase 4: Validation & Hardening",
            description="Testing and quality assurance",
            roles=[
                Role("QA Architect", "Global testing strategy",
                     ["Test strategy", "Test plans", "Coverage reports"]),
                Role("SDET", "Automated test code",
                     ["Test automation", "Framework maintenance", "CI integration"]),
                Role("Performance/Load Engineer", "Scale testing",
                     ["Load test scripts", "Performance baselines", "Capacity reports"]),
                Role("Manual QA / UAT Coordinator", "User acceptance testing",
                     ["Test cases", "UAT coordination", "Sign-off reports"]),
            ],
            exit_criteria=[
                "Test coverage requirements met",
                "Performance benchmarks achieved",
                "UAT sign-off obtained"
            ]
        ),
        # Phase 5: Delivery & Sustainment
        11: Team(
            id=11,
            name="Site Reliability Engineering (SRE)",
            phase="Phase 5: Delivery & Sustainment",
            description="Reliability and observability",
            roles=[
                Role("SRE Lead", "Error budget and uptime SLA",
                     ["SLOs", "Error budgets", "Reliability reports"]),
                Role("Observability Engineer", "Monitoring and logging",
                     ["Dashboards", "Alerts", "Log aggregation", "Traces"]),
                Role("Chaos Engineer", "Resiliency testing",
                     ["Chaos experiments", "Failure scenarios", "Recovery tests"]),
                Role("Incident Manager", "War room leadership",
                     ["Incident response", "Post-mortems", "Runbook updates"]),
            ],
            exit_criteria=[
                "Monitoring in place",
                "Alerts configured",
                "Runbooks complete",
                "Error budget healthy"
            ]
        ),
        12: Team(
            id=12,
            name="IT Operations & Support (NOC)",
            phase="Phase 5: Delivery & Sustainment",
            description="Production operations",
            roles=[
                Role("NOC Analyst", "24/7 monitoring",
                     ["Monitoring dashboards", "Alert triage", "Incident tickets"]),
                Role("Change Manager", "Deployment approval",
                     ["Change requests", "Deployment windows", "CAB approval"]),
                Role("Release Manager", "Go/No-Go coordination",
                     ["Release plans", "Rollback procedures", "Coordination"]),
                Role("L3 Support Engineer", "Production bug escalation",
                     ["Root cause analysis", "Hotfix coordination", "KB articles"]),
            ],
            exit_criteria=[
                "Change approved",
                "Release deployed",
                "Support handoff complete"
            ]
        ),
    }

    def __init__(self, project_name: str, config_path: Path = None, user_context: Optional[UserContext] = None, logger: Optional[StructuredLogger] = None):
        self.project_name = project_name
        self.teams: Dict[int, Team] = {}
        self.config_path = config_path or Path(f".teams/{project_name}.json")
        self.lock_path = Path(f".teams/{project_name}.lock")
        self.user_context = user_context
        self.logger = logger or StructuredLogger("team_manager")

    def _require_auth(self, operation: str, team_id: Optional[int] = None) -> None:
        """Check if user is authorized for the operation."""
        if self.user_context is None:
            raise PermissionDenied(f"Authentication required for {operation}")

        if not self.user_context.has_permission("team-lead"):
            raise PermissionDenied(
                f"User '{self.user_context.user_id}' with role '{self.user_context.role}' "
                f"does not have permission to {operation}"
            )

        if team_id is not None and not self.user_context.can_modify_team(team_id):
            raise PermissionDenied(
                f"User '{self.user_context.user_id}' cannot modify team {team_id}. "
                f"Requires admin role or team-lead for this specific team."
            )

    def initialize_project(self) -> None:
        """Initialize a new project with all teams.

        Requires admin role.
        """
        self._require_auth("initialize project")
        self.teams = {team_id: team for team_id, team in self.STANDARD_TEAMS.items()}
        self.save()
        print(f"✅ Initialized project '{self.project_name}' with {len(self.teams)} teams")

    def load(self) -> bool:
        """Load team configuration from disk with file locking.

        Uses shared lock to allow concurrent reads while preventing
        reads during writes.
        """
        if not self.config_path.exists():
            return False

        # Create lock file if it doesn't exist
        self.lock_path.parent.mkdir(parents=True, exist_ok=True)

        with FileLock(self.lock_path):
            with open(self.config_path, 'r') as f:
                # Use shared lock for reads
                fcntl.flock(f.fileno(), fcntl.LOCK_SH)
                try:
                    data = json.load(f)
                finally:
                    fcntl.flock(f.fileno(), fcntl.LOCK_UN)

        self.teams = {}
        for team_data in data.get("teams", []):
            team = Team(**team_data)
            team.roles = [Role(**r) for r in team_data.get("roles", [])]
            self.teams[team.id] = team

        return True

    def save(self) -> None:
        """Save team configuration to disk."""
        self.logger.info("config_save_start", {"config_path": str(self.config_path)})
        self.config_path.parent.mkdir(parents=True, exist_ok=True)

        data = {
            "project_name": self.project_name,
            "updated_at": datetime.now().isoformat(),
            "teams": [asdict(team) for team in self.teams.values()]
        }

        try:
            # Use file locking to prevent race conditions (SEC-004)
            with FileLock(self.lock_path):
                # Atomic write: write to temp file, then rename
                fd, temp_path = tempfile.mkstemp(
                    dir=self.config_path.parent,
                    prefix=f".{self.project_name}.tmp."
                )
                try:
                    with os.fdopen(fd, 'w') as f:
                        json.dump(data, f, indent=2)
                        f.flush()
                        os.fsync(f.fileno())

                    # Atomic rename
                    os.replace(temp_path, self.config_path)
                except Exception:
                    # Clean up temp file on error
                    try:
                        os.unlink(temp_path)
                    except FileNotFoundError:
                        pass
                    raise

            self.logger.info("config_saved", {
                "config_path": str(self.config_path),
                "team_count": len(self.teams)
            })
        except Exception as e:
            self.logger.error("config_save_failed", {
                "config_path": str(self.config_path),
                "error": str(e)
            }, exc_info=True)
            raise

    def assign_role(self, team_id: int, role_name: str, assignee: str) -> bool:
        """Assign a person to a role.

        Requires team-lead role (for their team) or admin role.
        """
        self._require_auth("assign role", team_id)
        self.logger.info("role_assignment_start", {
            "team_id": team_id,
            "role_name": role_name,
            "assignee": assignee,
            "user_id": self.user_context.user_id if self.user_context else None
        })

        if team_id not in self.teams:
            self.logger.error("team_not_found", {"team_id": team_id})
            return False

        team = self.teams[team_id]
        for role in team.roles:
            if role.name == role_name:
                role.assigned_to = assignee
                self.save()
                self.logger.info("role_assigned", {
                    "team_id": team_id,
                    "team_name": team.name,
                    "role_name": role_name,
                    "assignee": assignee
                })
                return True

        self.logger.error("role_not_found", {
            "team_id": team_id,
            "team_name": team.name,
            "role_name": role_name
        })
        return False
        return False

    def unassign_role(self, team_id: int, role_name: str) -> bool:
        """Remove assignment from a role."""
        if team_id not in self.teams:
            print(f"❌ Team {team_id} not found")
            return False

        team = self.teams[team_id]
        for role in team.roles:
            if role.name == role_name:
                if role.assigned_to is None:
                    print(f"⚠️  Role '{role_name}' in {team.name} is already unassigned")
                    return False
                previous_assignee = role.assigned_to
                role.assigned_to = None
                self.save()
                print(f"✅ Unassigned {previous_assignee} from {role_name} in {team.name}")
                return True

        print(f"❌ Role '{role_name}' not found in {team.name}")
        return False

    def start_team(self, team_id: int) -> bool:
        """Mark a team as active."""
        self.logger.info("team_start_request", {"team_id": team_id})

        if team_id not in self.teams:
            self.logger.error("team_not_found", {"team_id": team_id})
            return False

        team = self.teams[team_id]
        team.status = "active"
        team.started_at = datetime.now().isoformat()
        self.save()
        self.logger.info("team_started", {
            "team_id": team_id,
            "team_name": team.name,
            "status": team.status,
            "started_at": team.started_at
        })
        return True

    def complete_team(self, team_id: int) -> bool:
        """Mark a team as completed."""
        self.logger.info("team_complete_request", {"team_id": team_id})

        if team_id not in self.teams:
            self.logger.error("team_not_found", {"team_id": team_id})
            return False

        team = self.teams[team_id]
        team.status = "completed"
        team.completed_at = datetime.now().isoformat()
        self.save()
        self.logger.info("team_completed", {
            "team_id": team_id,
            "team_name": team.name,
            "status": team.status,
            "completed_at": team.completed_at
        })
        return True

    def get_phase_status(self, phase: str) -> dict:
        """Get status summary for a phase."""
        phase_teams = [t for t in self.teams.values() if t.phase == phase]

        total = len(phase_teams)
        completed = len([t for t in phase_teams if t.status == "completed"])
        active = len([t for t in phase_teams if t.status == "active"])

        result = {
            "phase": phase,
            "total_teams": total,
            "completed": completed,
            "active": active,
            "not_started": total - completed - active,
            "progress_pct": (completed / total * 100) if total > 0 else 0
        }

        self.logger.debug("phase_status_queried", {"phase": phase, "status": result})
        return result

    def list_teams(self, phase: str = None) -> None:
        """Print all teams."""
        teams = self.teams.values()
        if phase:
            teams = [t for t in teams if t.phase == phase]

        team_list = []
        for team in sorted(teams, key=lambda t: (t.phase, t.id)):
            assigned_roles = sum(1 for r in team.roles if r.assigned_to)
            team_list.append({
                "id": team.id,
                "name": team.name,
                "phase": team.phase,
                "status": team.status,
                "assigned_count": assigned_roles,
                "total_roles": len(team.roles)
            })

        self.logger.info("teams_listed", {
            "phase_filter": phase,
            "team_count": len(team_list),
            "teams": team_list
        })

    def get_agent_team(self, agent_type: str) -> Optional[Team]:
        """Map agent type to appropriate team."""
        mapping = {
            "planner": 2,      # Enterprise Architecture
            "coder": 7,        # Core Feature Squad
            "reviewer": 10,    # Quality Engineering
            "security": 9,     # Cybersecurity
            "tester": 10,      # Quality Engineering
            "ops": 11,         # SRE
        }
        team_id = mapping.get(agent_type.lower())
        team = self.teams.get(team_id) if team_id else None

        self.logger.debug("agent_team_mapped", {
            "agent_type": agent_type,
            "team_id": team_id,
            "found": team is not None
        })
        return team

    def validate_team_size(self, team_id: Optional[int] = None) -> dict:
        """Validate team sizes meet 4-6 member requirement.

        Returns dict with validation results.
        """
        MIN_TEAM_SIZE = 4
        MAX_TEAM_SIZE = 6

        results = {
            "valid": True,
            "violations": [],
            "teams_checked": 0
        }

        teams_to_check = [self.teams[team_id]] if team_id else self.teams.values()

        for team in teams_to_check:
            results["teams_checked"] += 1
            assigned_count = sum(1 for role in team.roles if role.assigned_to)

            if assigned_count < MIN_TEAM_SIZE:
                results["valid"] = False
                results["violations"].append({
                    "team_id": team.id,
                    "team_name": team.name,
                    "issue": "undersized",
                    "assigned": assigned_count,
                    "required": MIN_TEAM_SIZE
                })
                self.logger.warn("team_undersized", {
                    "team_id": team.id,
                    "team_name": team.name,
                    "assigned": assigned_count,
                    "required": MIN_TEAM_SIZE
                })
            elif assigned_count > MAX_TEAM_SIZE:
                results["valid"] = False
                results["violations"].append({
                    "team_id": team.id,
                    "team_name": team.name,
                    "issue": "oversized",
                    "assigned": assigned_count,
                    "maximum": MAX_TEAM_SIZE
                })
                self.logger.warn("team_oversized", {
                    "team_id": team.id,
                    "team_name": team.name,
                    "assigned": assigned_count,
                    "maximum": MAX_TEAM_SIZE
                })

        if results["valid"]:
            self.logger.info("team_size_validation_passed", {
                "teams_checked": results["teams_checked"]
            })
        else:
            self.logger.warn("team_size_validation_failed", {
                "teams_checked": results["teams_checked"],
                "violation_count": len(results["violations"]),
                "violations": results["violations"]
            })

        return results

    def delete_team(self, team_id: int, confirmed: bool = False) -> dict:
        """Delete a specific team from the project.

        Args:
            team_id: The ID of the team to delete
            confirmed: Whether deletion is confirmed (safety check)

        Returns:
            dict with deletion result
        """
        result = {
            "success": False,
            "team_id": team_id,
            "message": "",
            "requires_confirmation": False
        }

        if team_id not in self.teams:
            result["message"] = f"❌ Team {team_id} not found in project '{self.project_name}'"
            return result

        team = self.teams[team_id]

        if not confirmed:
            result["requires_confirmation"] = True
            result["message"] = (
                f"⚠️  Deletion requires confirmation. "
                f"Team {team_id} ({team.name}) will be permanently removed. "
                f"Set confirmed=true to proceed."
            )
            return result

        # Safety check: log deletion for audit
        deleted_team_name = team.name
        del self.teams[team_id]
        self.save()

        # Log audit information to stderr
        print(f"AUDIT: Team {team_id} ({deleted_team_name}) deleted from project '{self.project_name}' at {datetime.now().isoformat()}", file=sys.stderr)

        result["success"] = True
        result["message"] = f"✅ Team {team_id} ({deleted_team_name}) deleted from project '{self.project_name}'"
        return result

    def delete_project(self, confirmed: bool = False) -> dict:
        """Delete the entire project.

        Args:
            confirmed: Whether deletion is confirmed (safety check)

        Returns:
            dict with deletion result
        """
        result = {
            "success": False,
            "project_name": self.project_name,
            "message": "",
            "requires_confirmation": False
        }

        if not self.config_path.exists():
            result["message"] = f"❌ Project '{self.project_name}' not found"
            return result

        if not confirmed:
            result["requires_confirmation"] = True
            team_count = len(self.teams)
            result["message"] = (
                f"⚠️  Deletion requires confirmation. "
                f"Project '{self.project_name}' with {team_count} team(s) will be permanently deleted. "
                f"Set confirmed=true to proceed."
            )
            return result

        # Safety check: log deletion for audit
        team_count = len(self.teams)
        deletion_time = datetime.now().isoformat()

        # Delete the project file
        try:
            self.config_path.unlink()
            print(f"AUDIT: Project '{self.project_name}' ({team_count} teams) deleted at {deletion_time}", file=sys.stderr)

            result["success"] = True
            result["message"] = f"✅ Project '{self.project_name}' ({team_count} teams) deleted successfully"
        except Exception as e:
            result["message"] = f"❌ Error deleting project: {e}"

        return result


def main():
    parser = argparse.ArgumentParser(description="Team Manager - Standardized Team Layout")
    parser.add_argument("--project", required=True, help="Project name")
    parser.add_argument("--request-id", help="Correlation ID for request tracing")

    subparsers = parser.add_subparsers(dest="command", help="Command to run")

    # Init command
    init_parser = subparsers.add_parser("init", help="Initialize new project")

    # List command
    list_parser = subparsers.add_parser("list", help="List teams")
    list_parser.add_argument("--phase", help="Filter by phase")

    # Assign command
    assign_parser = subparsers.add_parser("assign", help="Assign person to role")
    assign_parser.add_argument("--team", type=int, required=True, help="Team ID")
    assign_parser.add_argument("--role", required=True, help="Role name")
    assign_parser.add_argument("--person", required=True, help="Person name")

    # Unassign command
    unassign_parser = subparsers.add_parser("unassign", help="Remove person from role")
    unassign_parser.add_argument("--team", type=int, required=True, help="Team ID")
    unassign_parser.add_argument("--role", required=True, help="Role name")

    # Start command
    start_parser = subparsers.add_parser("start", help="Start a team")
    start_parser.add_argument("--team", type=int, required=True, help="Team ID")

    # Complete command
    complete_parser = subparsers.add_parser("complete", help="Complete a team")
    complete_parser.add_argument("--team", type=int, required=True, help="Team ID")

    # Status command
    status_parser = subparsers.add_parser("status", help="Show phase status")
    status_parser.add_argument("--phase", help="Phase name")

    # Validate-size command
    validate_size_parser = subparsers.add_parser("validate-size", help="Validate team sizes (4-6 members)")
    validate_size_parser.add_argument("--team", type=int, help="Specific team ID to validate (optional)")

    # Delete-team command
    delete_team_parser = subparsers.add_parser("delete-team", help="Delete a specific team from the project")
    delete_team_parser.add_argument("--team", type=int, required=True, help="Team ID to delete")
    delete_team_parser.add_argument("--confirmed", action="store_true", help="Confirm deletion (required)")

    # Delete-project command
    delete_project_parser = subparsers.add_parser("delete-project", help="Delete the entire project")
    delete_project_parser.add_argument("--confirmed", action="store_true", help="Confirm deletion (required)")

    args = parser.parse_args()

    # Validate project name to prevent command injection
    validate_project_name(args.project)

    # Create logger for main and generate request_id if not provided
    request_id = args.request_id or f"tm-{datetime.utcnow().strftime('%Y%m%d%H%M%S')}-{os.getpid()}"
    cli_logger = StructuredLogger("team_manager_cli", request_id)
    cli_logger.info("cli_start", {"command": args.command, "project": args.project})

    manager = TeamManager(args.project, request_id=request_id)

    if args.command == "init":
        manager.initialize_project()
        print(f"\nTeams configuration saved to: {manager.config_path}")

    elif args.command in ["list", "assign", "unassign", "start", "complete", "status", "validate-size", "delete-team", "delete-project"]:
        if args.command in ["delete-team", "delete-project"]:
            # For delete commands, project may not exist yet (delete-project)
            if args.command == "delete-team" and not manager.load():
                print(f"❌ Project '{args.project}' not found.")
                sys.exit(1)
            if args.command == "delete-project":
                # Try to load but don't fail if file doesn't exist
                manager.load()
        else:
            if not manager.load():
                print(f"❌ Project '{args.project}' not found. Run: team_manager.py --project {args.project} init")
                sys.exit(1)

        if args.command == "list":
            manager.list_teams(args.phase)

        elif args.command == "assign":
            manager.assign_role(args.team, args.role, args.person)

        elif args.command == "unassign":
            manager.unassign_role(args.team, args.role)

        elif args.command == "start":
            manager.start_team(args.team)

        elif args.command == "complete":
            manager.complete_team(args.team)

        elif args.command == "status":
            if args.phase:
                status = manager.get_phase_status(args.phase)
                print(f"\n{status['phase']}")
                print(f"  Progress: {status['progress_pct']:.0f}%")
                print(f"  Teams: {status['completed']}/{status['total_teams']} complete")
                print(f"  Active: {status['active']}, Not started: {status['not_started']}")
            else:
                # Show all phases
                phases = set(t.phase for t in manager.teams.values())
                for phase in sorted(phases, key=lambda p: p.split(":")[0]):
                    status = manager.get_phase_status(phase)
                    print(f"\n{status['phase']}: {status['progress_pct']:.0f}% complete")

        elif args.command == "validate-size":
            results = manager.validate_team_size(args.team)
            if results["valid"]:
                print(f"✅ All {results['teams_checked']} teams have valid size (4-6 members)")
                sys.exit(0)
            else:
                print(f"❌ Team size violations found:")
                for violation in results["violations"]:
                    print(f"   {violation['message']}")
                sys.exit(1)

        elif args.command == "delete-team":
            result = manager.delete_team(args.team, confirmed=args.confirmed)
            print(result["message"])
            if not result["success"] and not result["requires_confirmation"]:
                sys.exit(1)

        elif args.command == "delete-project":
            result = manager.delete_project(confirmed=args.confirmed)
            print(result["message"])
            if not result["success"] and not result["requires_confirmation"]:
                sys.exit(1)

    else:
        parser.print_help()


if __name__ == "__main__":
    main()
