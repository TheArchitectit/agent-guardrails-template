#!/usr/bin/env python3
"""
Team Manager - Standardized Team Layout Manager

Manages team assignments, tracks phase progress, and validates
team composition against standardized enterprise layout.
"""

import argparse
import fcntl
import gzip
import json
import os
import re
import sys
import tempfile
import traceback
from dataclasses import dataclass, asdict
from datetime import datetime
from pathlib import Path
from typing import Any, List, Optional, Dict

# FUNC-005: Batch operations support
try:
    from .batch_operations import (
        import_csv, export_csv, import_json, export_json,
        create_csv_template, create_json_template
    )
except ImportError:
    from batch_operations import (
        import_csv, export_csv, import_json, export_json,
        create_csv_template, create_json_template
    )



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


# Valid phases from TEAM_STRUCTURE.md
VALID_PHASES = {
    "Phase 1: Strategy, Governance & Planning",
    "Phase 2: Platform & Foundation",
    "Phase 3: The Build Squads",
    "Phase 4: Validation & Hardening",
    "Phase 5: Delivery & Sustainment",
}

# Valid role names from TEAM_STRUCTURE.md (48 roles across 12 teams)
VALID_ROLES = {
    # Team 1: Business & Product Strategy
    "Business Relationship Manager",
    "Lead Product Manager",
    "Business Systems Analyst",
    "Financial Controller (FinOps)",
    # Team 2: Enterprise Architecture
    "Chief Architect",
    "Domain Architect",
    "Solution Architect",
    "Standards Lead",
    # Team 3: GRC
    "Compliance Officer",
    "Internal Auditor",
    "Privacy Engineer",
    "Policy Manager",
    # Team 4: Infrastructure & Cloud Ops
    "Cloud Architect",
    "IaC Engineer",
    "Network Security Engineer",
    "Storage Engineer",
    # Team 5: Platform Engineering
    "Platform Product Manager",
    "CI/CD Architect",
    "Kubernetes Administrator",
    "Developer Advocate",
    # Team 6: Data Governance & Analytics
    "Data Architect",
    "DBA",
    "Data Privacy Officer",
    "ETL Developer",
    # Team 7: Core Feature Squad
    "Technical Lead",
    "Senior Backend Engineer",
    "Senior Frontend Engineer",
    "Accessibility (A11y) Expert",
    "Technical Writer",
    # Team 8: Middleware & Integration
    "API Product Manager",
    "Integration Engineer",
    "Messaging Engineer",
    "IAM Specialist",
    # Team 9: Cybersecurity
    "Security Architect",
    "Vulnerability Researcher",
    "Penetration Tester",
    "DevSecOps Engineer",
    # Team 10: Quality Engineering
    "QA Architect",
    "SDET",
    "Performance/Load Engineer",
    "Manual QA / UAT Coordinator",
    # Team 11: SRE
    "SRE Lead",
    "Observability Engineer",
    "Chaos Engineer",
    "Incident Manager",
    # Team 12: IT Operations & Support
    "NOC Analyst",
    "Change Manager",
    "Release Manager",
    "L3 Support Engineer",
}


def validate_phase(phase: str) -> None:
    """Validate phase name against valid phases.

    Args:
        phase: Phase name to validate

    Raises:
        ValueError: If phase is not a valid phase name
    """
    if not phase:
        raise ValueError("phase is required")
    if phase not in VALID_PHASES:
        raise ValueError(
            f"Invalid phase: '{phase}'. Must be one of: "
            f"{', '.join(sorted(VALID_PHASES))}"
        )


def validate_role_name(role_name: str) -> None:
    """Validate role name against valid roles from TEAM_STRUCTURE.md.

    Args:
        role_name: Role name to validate

    Raises:
        ValueError: If role_name is not a valid role
    """
    if not role_name:
        raise ValueError("role_name is required")
    if len(role_name) > 128:
        raise ValueError("role_name must be 128 characters or less")
    # Check for control characters
    if re.search(r'[\x00-\x1f\x7f]', role_name):
        raise ValueError("role_name contains invalid control characters")
    if role_name not in VALID_ROLES:
        raise ValueError(
            f"Invalid role_name: '{role_name}'. Must be one of the 48 defined roles. "
            f"See TEAM_STRUCTURE.md for valid role definitions."
        )


def validate_person_name(person: str) -> None:
    """Validate person/assignee name format.

    Accepts email addresses or usernames with alphanumeric characters,
    hyphens, underscores, and dots.

    Args:
        person: Person name/identifier to validate

    Raises:
        ValueError: If person format is invalid
    """
    if not person:
        raise ValueError("person is required")
    if len(person) > 256:
        raise ValueError("person must be 256 characters or less")
    # Check for control characters
    if re.search(r'[\x00-\x1f\x7f]', person):
        raise ValueError("person contains invalid control characters")
    # Allow email format or username format
    # Email: user@domain.com
    # Username: alphanumeric + hyphens + underscores + dots
    email_pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
    username_pattern = r'^[a-zA-Z0-9_.-]+$'
    if not re.match(email_pattern, person) and not re.match(username_pattern, person):
        raise ValueError(
            f"Invalid person format: '{person}'. "
            f"Must be a valid email address or username (alphanumeric, dots, hyphens, underscores)"
        )


class PermissionDenied(Exception):
    """Raised when user lacks permission for an operation."""
    pass


class FileLockError(Exception):
    """Raised when file locking fails."""
    pass


class BackupManager:
    """Manages automatic backups of team configurations.

    Implements OPS-004: Automated backup before writes with versioning.
    Keeps last N versions and stores in .teams/backups/
    """

    DEFAULT_MAX_BACKUPS = 10

    def __init__(self, project_name: str, backup_dir: Path = None, max_backups: int = None):
        self.project_name = project_name
        self.backup_dir = backup_dir or Path(".teams/backups")
        self.max_backups = max_backups or self.DEFAULT_MAX_BACKUPS
        self.backup_dir.mkdir(parents=True, exist_ok=True)

    def _get_backup_path(self, timestamp: str = None) -> Path:
        """Generate backup file path with timestamp."""
        ts = timestamp or datetime.now().strftime("%Y%m%d_%H%M%S")
        return self.backup_dir / f"{self.project_name}_{ts}.json.gz"

    def create_backup(self, config_path: Path) -> Optional[Path]:
        """Create a backup of the current configuration.

        Args:
            config_path: Path to the configuration file to backup

        Returns:
            Path to the backup file, or None if no file exists to backup
        """
        if not config_path.exists():
            return None

        # Generate timestamp for this backup
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S_%f")
        backup_path = self._get_backup_path(timestamp)

        # Copy and compress the file
        try:
            import gzip
            with open(config_path, 'rb') as src:
                with gzip.open(backup_path, 'wb') as dst:
                    dst.write(src.read())

            # Clean up old backups
            self._cleanup_old_backups()

            return backup_path
        except Exception as e:
            # If backup fails, log but don't block the save
            print(f"⚠️  Backup creation failed: {e}", file=sys.stderr)
            return None

    def _cleanup_old_backups(self) -> None:
        """Remove oldest backups keeping only max_backups versions."""
        try:
            backups = sorted(
                self.backup_dir.glob(f"{self.project_name}_*.json.gz"),
                key=lambda p: p.stat().st_mtime
            )

            while len(backups) > self.max_backups:
                oldest = backups.pop(0)
                try:
                    oldest.unlink()
                    print(f"🗑️  Removed old backup: {oldest.name}", file=sys.stderr)
                except OSError:
                    pass
        except Exception:
            pass

    def list_backups(self) -> List[Dict[str, Any]]:
        """List all available backups for this project.

        Returns:
            List of dicts with backup info: path, timestamp, size
        """
        backups = []
        for backup_file in sorted(self.backup_dir.glob(f"{self.project_name}_*.json.gz"), reverse=True):
            try:
                stat = backup_file.stat()
                # Extract timestamp from filename
                timestamp_str = backup_file.stem.replace(f"{self.project_name}_", "")
                backups.append({
                    "path": str(backup_file),
                    "filename": backup_file.name,
                    "timestamp": timestamp_str,
                    "size_bytes": stat.st_size,
                    "created_at": datetime.fromtimestamp(stat.st_mtime).isoformat()
                })
            except OSError:
                continue
        return backups

    def restore_backup(self, backup_path: Path, target_path: Path) -> bool:
        """Restore a backup to the target path.

        Args:
            backup_path: Path to the backup file
            target_path: Path to restore to

        Returns:
            True if successful, False otherwise
        """
        try:
            import gzip
            with gzip.open(backup_path, 'rb') as src:
                content = src.read()

            # Atomic restore: write to temp then rename
            fd, temp_path = tempfile.mkstemp(
                dir=target_path.parent,
                prefix=f".{self.project_name}.restore.tmp."
            )
            try:
                with os.fdopen(fd, 'wb') as f:
                    f.write(content)
                    f.flush()
                    os.fsync(f.fileno())
                os.replace(temp_path, target_path)
                return True
            except Exception:
                try:
                    os.unlink(temp_path)
                except FileNotFoundError:
                    pass
                raise
        except Exception as e:
            print(f"❌ Restore failed: {e}", file=sys.stderr)
            return False


class AuditLogger:
    """Audit logging for team operations (SEC-008).

    Logs all team modifications with user, timestamp, action, and before/after state.
    Stores in .teams/audit.log
    """

    def __init__(self, project_name: str, audit_dir: Path = None):
        self.project_name = project_name
        self.audit_dir = audit_dir or Path(".teams")
        self.audit_file = self.audit_dir / "audit.log"
        self.audit_dir.mkdir(parents=True, exist_ok=True)

    def _get_user_id(self, user_context: Optional["UserContext"]) -> str:
        """Extract user ID from context or return 'system'."""
        if user_context:
            return user_context.user_id
        return "system"

    def log_action(self, action: str, details: Dict[str, Any], user_context: Optional["UserContext"] = None) -> None:
        """Log an audit action.

        Args:
            action: The action performed (e.g., 'assign_role', 'start_team')
            details: Dict containing before/after state and other details
            user_context: Optional user context for RBAC info
        """
        entry = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "project": self.project_name,
            "user": self._get_user_id(user_context),
            "role": user_context.role if user_context else "system",
            "action": action,
            "details": details
        }

        try:
            with open(self.audit_file, 'a') as f:
                fcntl.flock(f.fileno(), fcntl.LOCK_EX)
                try:
                    f.write(json.dumps(entry) + "\n")
                    f.flush()
                    os.fsync(f.fileno())
                finally:
                    fcntl.flock(f.fileno(), fcntl.LOCK_UN)
        except Exception as e:
            print(f"⚠️  Audit logging failed: {e}", file=sys.stderr)

    def query_audit_log(self, start_time: Optional[datetime] = None,
                        end_time: Optional[datetime] = None,
                        user: Optional[str] = None,
                        action: Optional[str] = None,
                        team_id: Optional[int] = None,
                        limit: int = 100) -> List[Dict[str, Any]]:
        """Query the audit log with filters.

        Args:
            start_time: Optional start time filter
            end_time: Optional end time filter
            user: Optional user filter
            action: Optional action filter
            team_id: Optional team_id filter
            limit: Maximum number of entries to return

        Returns:
            List of matching audit entries
        """
        if not self.audit_file.exists():
            return []

        results = []
        try:
            with open(self.audit_file, 'r') as f:
                for line in f:
                    if not line.strip():
                        continue
                    try:
                        entry = json.loads(line)

                        # Apply filters
                        if start_time:
                            entry_time = datetime.fromisoformat(entry["timestamp"].replace("Z", "+00:00"))
                            if entry_time < start_time:
                                continue
                        if end_time:
                            entry_time = datetime.fromisoformat(entry["timestamp"].replace("Z", "+00:00"))
                            if entry_time > end_time:
                                continue
                        if user and entry.get("user") != user:
                            continue
                        if action and entry.get("action") != action:
                            continue
                        if team_id is not None:
                            entry_team_id = entry.get("details", {}).get("team_id")
                            if entry_team_id != team_id:
                                continue

                        results.append(entry)

                        if len(results) >= limit:
                            break
                    except json.JSONDecodeError:
                        continue
        except Exception as e:
            print(f"⚠️  Audit query failed: {e}", file=sys.stderr)

        return results

    def get_recent_actions(self, count: int = 10) -> List[Dict[str, Any]]:
        """Get the most recent audit actions.

        Args:
            count: Number of entries to return

        Returns:
            List of recent audit entries
        """
        return self.query_audit_log(limit=count)


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

    def __init__(self, project_name: str, config_path: Path = None, user_context: Optional["UserContext"] = None, logger: Optional[StructuredLogger] = None, enable_backup: bool = True, enable_audit: bool = True, max_backups: int = 10):
        self.project_name = project_name
        self.teams: Dict[int, Team] = {}
        self.config_path = config_path or Path(f".teams/{project_name}.json")
        self.lock_path = Path(f".teams/{project_name}.lock")
        self.user_context = user_context
        self.logger = logger or StructuredLogger("team_manager")

        # OPS-004: Backup manager
        self.enable_backup = enable_backup
        if enable_backup:
            self.backup_manager = BackupManager(project_name, max_backups=max_backups)
        else:
            self.backup_manager = None

        # SEC-008: Audit logger
        self.enable_audit = enable_audit
        if enable_audit:
            self.audit_logger = AuditLogger(project_name)
        else:
            self.audit_logger = None

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
        """Save team configuration to disk.

        OPS-004: Creates automatic backup before saving.
        """
        self.logger.info("config_save_start", {"config_path": str(self.config_path)})
        self.config_path.parent.mkdir(parents=True, exist_ok=True)

        # OPS-004: Create backup before write if config exists
        backup_path = None
        if self.enable_backup and self.backup_manager and self.config_path.exists():
            backup_path = self.backup_manager.create_backup(self.config_path)
            if backup_path:
                self.logger.info("backup_created", {"backup_path": str(backup_path)})

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
                "team_count": len(self.teams),
                "backup_path": str(backup_path) if backup_path else None
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
        SEC-008: Logs audit trail.
        """
        # Validate inputs (FUNC-003, SEC-002, SEC-003)
        try:
            validate_role_name(role_name)
            validate_person_name(assignee)
        except ValueError as e:
            self.logger.error("role_assignment_validation_failed", {
                "team_id": team_id,
                "role_name": role_name,
                "assignee": assignee,
                "error": str(e)
            })
            print(f"❌ Validation error: {e}", file=sys.stderr)
            return False

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
                # SEC-008: Capture before state for audit
                previous_assignee = role.assigned_to
                role.assigned_to = assignee
                self.save()

                # SEC-008: Log audit trail
                if self.enable_audit and self.audit_logger:
                    self.audit_logger.log_action(
                        "assign_role",
                        {
                            "team_id": team_id,
                            "team_name": team.name,
                            "role_name": role_name,
                            "before": previous_assignee,
                            "after": assignee
                        },
                        self.user_context
                    )

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

    def unassign_role(self, team_id: int, role_name: str) -> bool:
        """Remove assignment from a role.

        SEC-008: Logs audit trail.
        """
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

                # SEC-008: Log audit trail
                if self.enable_audit and self.audit_logger:
                    self.audit_logger.log_action(
                        "unassign_role",
                        {
                            "team_id": team_id,
                            "team_name": team.name,
                            "role_name": role_name,
                            "before": previous_assignee,
                            "after": None
                        },
                        self.user_context
                    )

                print(f"✅ Unassigned {previous_assignee} from {role_name} in {team.name}")
                return True

        print(f"❌ Role '{role_name}' not found in {team.name}")
        return False

    def start_team(self, team_id: int) -> bool:
        """Mark a team as active.

        SEC-008: Logs audit trail.
        """
        self.logger.info("team_start_request", {"team_id": team_id})

        if team_id not in self.teams:
            self.logger.error("team_not_found", {"team_id": team_id})
            return False

        team = self.teams[team_id]
        previous_status = team.status
        team.status = "active"
        team.started_at = datetime.now().isoformat()
        self.save()

        # SEC-008: Log audit trail
        if self.enable_audit and self.audit_logger:
            self.audit_logger.log_action(
                "start_team",
                {
                    "team_id": team_id,
                    "team_name": team.name,
                    "before": previous_status,
                    "after": "active",
                    "started_at": team.started_at
                },
                self.user_context
            )

        self.logger.info("team_started", {
            "team_id": team_id,
            "team_name": team.name,
            "status": team.status,
            "started_at": team.started_at
        })
        return True

    def complete_team(self, team_id: int) -> bool:
        """Mark a team as completed.

        SEC-008: Logs audit trail.
        """
        self.logger.info("team_complete_request", {"team_id": team_id})

        if team_id not in self.teams:
            self.logger.error("team_not_found", {"team_id": team_id})
            return False

        team = self.teams[team_id]
        previous_status = team.status
        team.status = "completed"
        team.completed_at = datetime.now().isoformat()
        self.save()

        # SEC-008: Log audit trail
        if self.enable_audit and self.audit_logger:
            self.audit_logger.log_action(
                "complete_team",
                {
                    "team_id": team_id,
                    "team_name": team.name,
                    "before": previous_status,
                    "after": "completed",
                    "completed_at": team.completed_at
                },
                self.user_context
            )

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
        # Validate phase filter if provided (FUNC-004)
        if phase is not None:
            try:
                validate_phase(phase)
            except ValueError as e:
                self.logger.error("list_teams_validation_failed", {"error": str(e)})
                print(f"❌ Validation error: {e}", file=sys.stderr)
                return

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

        # Capture team data before deletion for audit
        team_data = asdict(team)
        deleted_team_name = team.name
        del self.teams[team_id]
        self.save()

        # SEC-008: Log audit trail
        if self.enable_audit and self.audit_logger:
            self.audit_logger.log_action(
                "delete_team",
                {
                    "team_id": team_id,
                    "team_name": deleted_team_name,
                    "deleted_data": team_data
                },
                self.user_context
            )

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

        # Capture project data before deletion for audit
        team_count = len(self.teams)
        deletion_time = datetime.now().isoformat()

        # SEC-008: Log audit trail before deletion
        if self.enable_audit and self.audit_logger:
            self.audit_logger.log_action(
                "delete_project",
                {
                    "project_name": self.project_name,
                    "team_count": team_count,
                    "teams": [asdict(team) for team in self.teams.values()],
                    "deleted_at": deletion_time
                },
                self.user_context
            )

        # Delete the project file
        try:
            self.config_path.unlink()

            result["success"] = True
            result["message"] = f"✅ Project '{self.project_name}' ({team_count} teams) deleted successfully"
        except Exception as e:
            result["message"] = f"❌ Error deleting project: {e}"

        return result

    def list_backups(self) -> List[Dict[str, Any]]:
        """List all available backups for this project.

        Returns:
            List of backup info dicts
        """
        if self.backup_manager:
            return self.backup_manager.list_backups()
        return []

    def restore_backup(self, backup_filename: str) -> dict:
        """Restore from a backup file.

        Args:
            backup_filename: Name of the backup file to restore

        Returns:
            Dict with restore result
        """
        result = {
            "success": False,
            "message": "",
            "backup_file": backup_filename
        }

        if not self.backup_manager:
            result["message"] = "❌ Backup manager not enabled"
            return result

        backup_path = self.backup_manager.backup_dir / backup_filename
        if not backup_path.exists():
            result["message"] = f"❌ Backup file not found: {backup_filename}"
            return result

        # Create backup of current state before restore
        if self.config_path.exists():
            current_backup = self.backup_manager.create_backup(self.config_path)
            if current_backup:
                print(f"💾 Created pre-restore backup: {current_backup.name}", file=sys.stderr)

        # Perform restore
        if self.backup_manager.restore_backup(backup_path, self.config_path):
            # Reload the teams data
            self.load()

            # Log audit trail
            if self.enable_audit and self.audit_logger:
                self.audit_logger.log_action(
                    "restore_backup",
                    {
                        "backup_file": backup_filename,
                        "restored_path": str(self.config_path),
                        "team_count": len(self.teams)
                    },
                    self.user_context
                )

            result["success"] = True
            result["message"] = f"✅ Successfully restored from {backup_filename}"
            result["team_count"] = len(self.teams)
        else:
            result["message"] = f"❌ Failed to restore from {backup_filename}"

        return result

    def query_audit(self, **filters) -> List[Dict[str, Any]]:
        """Query the audit log.

        Args:
            **filters: Optional filters like user, action, team_id, start_time, end_time, limit

        Returns:
            List of audit entries
        """
        if not self.audit_logger:
            return []
        return self.audit_logger.query_audit_log(**filters)

    def get_recent_audit(self, count: int = 10) -> List[Dict[str, Any]]:
        """Get recent audit entries.

        Args:
            count: Number of entries to return

        Returns:
            List of recent audit entries
        """
        if not self.audit_logger:
            return []
        return self.audit_logger.get_recent_actions(count)

    def health_check(self) -> dict:
        """Perform health check on team manager.

        OPS-003: Checks Python backend status and file system access.

        Returns:
            dict with health status information
        """
        health = {
            "status": "healthy",
            "checks": {},
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "version": "1.0.0"
        }

        # Check 1: Python environment
        try:
            import sys
            health["checks"]["python"] = {
                "status": "pass",
                "version": f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}"
            }
        except Exception as e:
            health["checks"]["python"] = {
                "status": "fail",
                "error": str(e)
            }
            health["status"] = "unhealthy"

        # Check 2: File system access
        try:
            import tempfile
            test_dir = Path(tempfile.gettempdir()) / "team_manager_health"
            test_dir.mkdir(exist_ok=True)
            test_file = test_dir / ".health_check"
            test_file.write_text("ok")
            content = test_file.read_text()
            test_file.unlink()
            test_dir.rmdir()

            if content == "ok":
                health["checks"]["filesystem"] = {
                    "status": "pass",
                    "writable": True
                }
            else:
                health["checks"]["filesystem"] = {
                    "status": "fail",
                    "error": "Read/write mismatch"
                }
                health["status"] = "unhealthy"
        except Exception as e:
            health["checks"]["filesystem"] = {
                "status": "fail",
                "error": str(e)
            }
            health["status"] = "unhealthy"

        # Check 3: Config directory access
        try:
            self.config_path.parent.mkdir(parents=True, exist_ok=True)
            health["checks"]["config_dir"] = {
                "status": "pass",
                "path": str(self.config_path.parent),
                "accessible": True
            }
        except Exception as e:
            health["checks"]["config_dir"] = {
                "status": "fail",
                "path": str(self.config_path.parent),
                "error": str(e)
            }
            health["status"] = "unhealthy"

        # Check 4: JSON serialization
        try:
            test_data = {"test": True, "timestamp": datetime.utcnow().isoformat()}
            json.dumps(test_data)
            health["checks"]["json"] = {
                "status": "pass"
            }
        except Exception as e:
            health["checks"]["json"] = {
                "status": "fail",
                "error": str(e)
            }
            health["status"] = "unhealthy"

        return health

    # FUNC-005: Batch operation wrappers
    def import_csv_file(self, csv_path: Path, dry_run: bool = False) -> Dict[str, Any]:
        """Import role assignments from CSV file."""
        return import_csv(self, csv_path, dry_run)

    def export_csv_file(self, csv_path: Path) -> Dict[str, Any]:
        """Export role assignments to CSV file."""
        return export_csv(self, csv_path)

    def import_json_file(self, json_path: Path, dry_run: bool = False) -> Dict[str, Any]:
        """Import role assignments from JSON file."""
        return import_json(self, json_path, dry_run)

    def export_json_file(self, json_path: Path, pretty: bool = True) -> Dict[str, Any]:
        """Export project state to JSON file."""
        return export_json(self, json_path, pretty)


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

    # List-backups command (OPS-004)
    list_backups_parser = subparsers.add_parser("list-backups", help="List available backups")

    # Restore command (OPS-004)
    restore_parser = subparsers.add_parser("restore", help="Restore from a backup")
    restore_parser.add_argument("--backup", required=True, help="Backup filename to restore")

    # Audit command (SEC-008)
    audit_parser = subparsers.add_parser("audit", help="Query audit log")
    audit_parser.add_argument("--user", help="Filter by user")
    audit_parser.add_argument("--action", help="Filter by action type")
    audit_parser.add_argument("--team", type=int, help="Filter by team ID")
    audit_parser.add_argument("--limit", type=int, default=20, help="Maximum entries to show (default: 20)")
    audit_parser.add_argument("--recent", action="store_true", help="Show most recent entries")

    # Health command (OPS-003)
    health_parser = subparsers.add_parser("health", help="Check team manager health status")

    # FUNC-005: Batch operations commands
    # Import CSV command
    import_csv_parser = subparsers.add_parser("import-csv", help="Import role assignments from CSV")
    import_csv_parser.add_argument("--file", required=True, help="Path to CSV file")
    import_csv_parser.add_argument("--dry-run", action="store_true", help="Validate without making changes")

    # Export CSV command
    export_csv_parser = subparsers.add_parser("export-csv", help="Export role assignments to CSV")
    export_csv_parser.add_argument("--file", required=True, help="Path to output CSV file")

    # Import JSON command
    import_json_parser = subparsers.add_parser("import-json", help="Import role assignments from JSON")
    import_json_parser.add_argument("--file", required=True, help="Path to JSON file")
    import_json_parser.add_argument("--dry-run", action="store_true", help="Validate without making changes")

    # Export JSON command
    export_json_parser = subparsers.add_parser("export-json", help="Export project state to JSON")
    export_json_parser.add_argument("--file", required=True, help="Path to output JSON file")
    export_json_parser.add_argument("--compact", action="store_true", help="Output compact JSON (no indentation)")

    # Template commands
    template_csv_parser = subparsers.add_parser("template-csv", help="Create CSV template for bulk assignments")
    template_csv_parser.add_argument("--file", default="assignments_template.csv", help="Output file path")

    template_json_parser = subparsers.add_parser("template-json", help="Create JSON template for bulk assignments")
    template_json_parser.add_argument("--file", default="assignments_template.json", help="Output file path")

    args = parser.parse_args()

    # Validate project name to prevent command injection
    validate_project_name(args.project)

    # Create logger for main and generate request_id if not provided
    request_id = args.request_id or f"tm-{datetime.utcnow().strftime('%Y%m%d%H%M%S')}-{os.getpid()}"
    cli_logger = StructuredLogger("team_manager_cli", request_id)
    cli_logger.info("cli_start", {"command": args.command, "project": args.project})

    manager = TeamManager(args.project)

    if args.command == "init":
        manager.initialize_project()
        print(f"\nTeams configuration saved to: {manager.config_path}")

    elif args.command in ["list", "assign", "unassign", "start", "complete", "status", "validate-size", "delete-team", "delete-project", "list-backups", "restore", "audit"]:
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

        elif args.command == "list-backups":
            backups = manager.list_backups()
            if backups:
                print(f"\n📦 Available backups for '{args.project}':")
                print(f"{'Filename':<50} {'Size':>10} {'Created At'}")
                print("-" * 90)
                for backup in backups:
                    size_kb = backup["size_bytes"] / 1024
                    print(f"{backup['filename']:<50} {size_kb:>9.1f}KB {backup['created_at']}")
                print(f"\nTotal backups: {len(backups)}")
            else:
                print(f"ℹ️  No backups found for '{args.project}'")

        elif args.command == "restore":
            result = manager.restore_backup(args.backup)
            print(result["message"])
            if result["success"]:
                print(f"\n📊 Project now has {result.get('team_count', 0)} team(s)")
            else:
                sys.exit(1)

        elif args.command == "audit":
            if args.recent:
                entries = manager.get_recent_audit(args.limit)
            else:
                filters = {"limit": args.limit}
                if args.user:
                    filters["user"] = args.user
                if args.action:
                    filters["action"] = args.action
                if args.team:
                    filters["team_id"] = args.team
                entries = manager.query_audit(**filters)

            if entries:
                print(f"\n📋 Audit log entries for '{args.project}':")
                print(f"{'Timestamp':<25} {'User':<15} {'Action':<20} {'Details'}")
                print("-" * 100)
                for entry in entries:
                    ts = entry["timestamp"].replace("T", " ").replace("Z", "")[:19]
                    user = entry.get("user", "unknown")[:14]
                    action = entry.get("action", "unknown")[:19]
                    details = json.dumps(entry.get("details", {}))[:50]
                    print(f"{ts:<25} {user:<15} {action:<20} {details}")
                print(f"\nTotal entries: {len(entries)}")
            else:
                print(f"ℹ️  No audit entries found for '{args.project}'")

    elif args.command == "health":
        # OPS-003: Health check - doesn't require project to exist
        health = manager.health_check()
        print(json.dumps(health, indent=2))
        if health["status"] != "healthy":
            sys.exit(1)

    else:
        parser.print_help()


if __name__ == "__main__":
    main()
