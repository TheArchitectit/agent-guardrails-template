#!/usr/bin/env python3
"""
Team Manager - Standardized Team Layout Manager

Manages team assignments, tracks phase progress, and validates
team composition against standardized enterprise layout.
"""

import argparse
import json
import sys
from dataclasses import dataclass, asdict
from datetime import datetime
from pathlib import Path
from typing import List, Optional, Dict


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

    def __init__(self, project_name: str, config_path: Path = None):
        self.project_name = project_name
        self.teams: Dict[int, Team] = {}
        self.config_path = config_path or Path(f".teams/{project_name}.json")

    def initialize_project(self) -> None:
        """Initialize a new project with all teams."""
        self.teams = {team_id: team for team_id, team in self.STANDARD_TEAMS.items()}
        self.save()
        print(f"✅ Initialized project '{self.project_name}' with {len(self.teams)} teams")

    def load(self) -> bool:
        """Load team configuration from disk."""
        if not self.config_path.exists():
            return False

        with open(self.config_path) as f:
            data = json.load(f)

        self.teams = {}
        for team_data in data.get("teams", []):
            team = Team(**team_data)
            team.roles = [Role(**r) for r in team_data.get("roles", [])]
            self.teams[team.id] = team

        return True

    def save(self) -> None:
        """Save team configuration to disk."""
        self.config_path.parent.mkdir(parents=True, exist_ok=True)

        data = {
            "project_name": self.project_name,
            "updated_at": datetime.now().isoformat(),
            "teams": [asdict(team) for team in self.teams.values()]
        }

        with open(self.config_path, 'w') as f:
            json.dump(data, f, indent=2)

    def assign_role(self, team_id: int, role_name: str, assignee: str) -> bool:
        """Assign a person to a role."""
        if team_id not in self.teams:
            print(f"❌ Team {team_id} not found")
            return False

        team = self.teams[team_id]
        for role in team.roles:
            if role.name == role_name:
                role.assigned_to = assignee
                self.save()
                print(f"✅ Assigned {assignee} to {role_name} in {team.name}")
                return True

        print(f"❌ Role '{role_name}' not found in {team.name}")
        return False

    def start_team(self, team_id: int) -> bool:
        """Mark a team as active."""
        if team_id not in self.teams:
            return False

        team = self.teams[team_id]
        team.status = "active"
        team.started_at = datetime.now().isoformat()
        self.save()
        print(f"🚀 Team {team_id} ({team.name}) is now active")
        return True

    def complete_team(self, team_id: int) -> bool:
        """Mark a team as completed."""
        if team_id not in self.teams:
            return False

        team = self.teams[team_id]

        # Check if all exit criteria are met
        # In practice, this would involve checking external systems
        team.status = "completed"
        team.completed_at = datetime.now().isoformat()
        self.save()
        print(f"✅ Team {team_id} ({team.name}) completed")
        return True

    def get_phase_status(self, phase: str) -> dict:
        """Get status summary for a phase."""
        phase_teams = [t for t in self.teams.values() if t.phase == phase]

        total = len(phase_teams)
        completed = len([t for t in phase_teams if t.status == "completed"])
        active = len([t for t in phase_teams if t.status == "active"])

        return {
            "phase": phase,
            "total_teams": total,
            "completed": completed,
            "active": active,
            "not_started": total - completed - active,
            "progress_pct": (completed / total * 100) if total > 0 else 0
        }

    def list_teams(self, phase: str = None) -> None:
        """Print all teams."""
        teams = self.teams.values()
        if phase:
            teams = [t for t in teams if t.phase == phase]

        current_phase = None
        for team in sorted(teams, key=lambda t: (t.phase, t.id)):
            if team.phase != current_phase:
                current_phase = team.phase
                print(f"\n{'='*60}")
                print(f"  {current_phase}")
                print(f"{'='*60}")

            status_icon = {
                "not_started": "⏳",
                "active": "🟢",
                "completed": "✅",
                "blocked": "🛑"
            }.get(team.status, "❓")

            print(f"\n{status_icon} Team {team.id}: {team.name}")
            print(f"   Description: {team.description}")
            print(f"   Status: {team.status}")

            print(f"\n   Roles:")
            for role in team.roles:
                assignee = role.assigned_to or "(unassigned)"
                print(f"      - {role.name}: {assignee}")

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
        return self.teams.get(team_id) if team_id else None


def main():
    parser = argparse.ArgumentParser(description="Team Manager - Standardized Team Layout")
    parser.add_argument("--project", required=True, help="Project name")

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

    # Start command
    start_parser = subparsers.add_parser("start", help="Start a team")
    start_parser.add_argument("--team", type=int, required=True, help="Team ID")

    # Complete command
    complete_parser = subparsers.add_parser("complete", help="Complete a team")
    complete_parser.add_argument("--team", type=int, required=True, help="Team ID")

    # Status command
    status_parser = subparsers.add_parser("status", help="Show phase status")
    status_parser.add_argument("--phase", help="Phase name")

    args = parser.parse_args()

    manager = TeamManager(args.project)

    if args.command == "init":
        manager.initialize_project()
        print(f"\nTeams configuration saved to: {manager.config_path}")

    elif args.command in ["list", "assign", "start", "complete", "status"]:
        if not manager.load():
            print(f"❌ Project '{args.project}' not found. Run: team_manager.py --project {args.project} init")
            sys.exit(1)

        if args.command == "list":
            manager.list_teams(args.phase)

        elif args.command == "assign":
            manager.assign_role(args.team, args.role, args.person)

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

    else:
        parser.print_help()


if __name__ == "__main__":
    main()
