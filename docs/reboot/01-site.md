# Agent Guardrails website: React design spec
Prepared Oct 1, 2026. Design and build spec only. Nothing is built, deployed or published. Domain, hosting and any paid service are undecided and need Roger's word.

## Purpose
Tell the reboot story honestly, show what Agent Guardrails does in under a minute, and get a developer from landing to a working install. The site must never promise protection the product does not provide.

## Stack (proposed defaults)
- Vite + React 18 + TypeScript, React Router, static export, deployable to any static host (GitHub Pages, Cloudflare Pages, Netlify). Pick at deploy time.
- Styling: CSS variables plus Tailwind or CSS Modules (pick one; default Tailwind).
- Content: MDX files in repo for story, docs excerpts, changelog. Install snippets and the host matrix are generated from repo data (tool manifest, compatibility matrix), not typed by hand.
- Motion: CSS and Framer Motion for small transitions only; honor prefers-reduced-motion.
- Analytics: none by default. If added later, privacy-respecting and disclosed.
- Tests: Vitest + Testing Library, Playwright smoke across 3 viewports, axe accessibility run in CI, Lighthouse budget.

## Site map
1. Home
2. Story ("Why we are rebooting")
3. How it works (advisory vs checked vs blocking)
4. Install (host tabs)
5. Compatibility (generated matrix)
6. Policies (Four Laws, packs)
7. Roadmap (milestones from the feature plan)
8. Docs (link out or embedded)
9. Safety and security (threat model, report channel)
10. Community (contributing, good first issues)

## Home page sections, in order
1. Hero. Headline candidates: "Boundaries for AI agents, with receipts." / "Move fast. Keep a way back." Sub: "An MCP server and editor extensions that check what your AI agent is about to do, and tell you plainly what was and was not enforced." Primary CTA: Install in 5 minutes. Secondary: Read our story.
2. The problem in one screen. Short, no fear-mongering: speed without boundaries turns small mistakes into big ones.
3. Honesty table (signature component, below). This is the visual identity of the site.
4. Four Laws as four cards: read before editing; stay in scope; verify before committing; halt when uncertain.
5. 60-second demo: video or looping captioned terminal clip of a blocked deletion (sanitized, see video script).
6. Works with: host logos only for hosts with a PASS row in the matrix; others shown as "planned".
7. Story teaser with link.
8. Install snippet tabs.
9. Footer: repo, license (BSD-3-Clause), security contact, honest-claims policy.

## Signature component: EnforcementTable
Three rows, always shown together wherever claims about protection appear:
- Advisory: "Your agent can ask. It can also skip."
- Checked operation: "The server verifies an operation it controls. A permit is tied to that exact action and expires."
- Blocking: "Only where a tested hook or wrapper intercepts and denies. Shown per host and version."
Each cell reads its status from generated data. Missing evidence renders "Not enforced", never a check mark. Unknown renders "Unknown", never green.

## Other components
- HostTabs (VS Code Copilot, Claude Code, Cursor, others as they pass tests), with copy buttons and version-tested badge with date; stale badges (older than 90 days) turn amber.
- VerdictBadge: PASS, FAIL, UNKNOWN, ERROR. UNKNOWN and ERROR are visually distinct from PASS, colour plus icon plus text.
- StoryTimeline: factual timeline limited to verified facts (see story kit review notes).
- RoadmapBoard: milestones M0-M6 with status read from repo issues.
- CodeBlock with redaction-safe examples only (placeholder values).
- NavBar with skip link, SiteFooter, ThemeToggle (light/dark/system).

## Visual design
- Tone: calm, serious, human. Not neon "cyber". Think safety-gear craftsmanship.
- Palette (proposed): ink #0F172A, paper #F8FAFC, signal amber #F59E0B (advisory), steel blue #2563EB (checked), deep green #15803D (blocking, used sparingly and only for verified), alert red #B91C1C (FAIL). Contrast checked to WCAG AA.
- Type: system UI stack plus one open-licensed display face (e.g. Inter or Source Serif for story). Body 17-18px, line length 65-75 characters.
- Layout: 12-column grid, generous whitespace, one idea per section.
- Imagery: original illustrations or diagrams only. No stock "hacker" imagery. No real terminals with hostnames, IPs or paths.

## Accessibility and performance
- WCAG 2.2 AA target. Keyboard-complete, visible focus, semantic landmarks, captions and transcript for every video, alt text, no color-only meaning.
- Lighthouse targets: performance 90+, accessibility 100, SEO 95+. LCP under 2.5s on mid-range mobile. Video lazy-loaded.
- Open Graph and social cards per page, generated at build.

## Content rules (public)
- Scrub: no hostnames, IPs, network topology, credential names or values, operator logs, internal tool names beyond what is public.
- No claims of users, certifications, hours spent, or "complete safety".
- Near-loss account is stated as Roger's experience; verified facts only: main reduced to a retirement notice, tag v3.7.1 kept the code.
- No statement that OpenClaw acted maliciously.
- Every enforcement claim links to a test or evidence record.

## Build plan (when approved)
1. Scaffold + tokens + layout (1 day)
2. EnforcementTable + generated data pipeline (2 days)
3. Home + Story + Install (3 days)
4. Compatibility + Roadmap + Policies (3 days)
5. A11y, perf, QA, cross-browser (2 days)
6. Review by Roger, then deploy to a preview URL before any public domain.
Rough total: about 2 weeks for one engineer; unvalidated estimate.

## Acceptance
- axe: zero serious/critical issues; keyboard walkthrough passes.
- Every claim on the site maps to a repo artifact (test, matrix row, doc).
- A clean-machine user reaches a working install following only the page, timed and recorded.
- Scrub scan finds zero hostnames, IPs, tokens, internal paths.
- Roger approves public copy before publication.

