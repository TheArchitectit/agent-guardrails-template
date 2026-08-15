# Vibe Coding Patterns

> Rapid development patterns for high-velocity AI sessions

**See also:** [Prompting Guide overview](prompting-guide.md)

---

These prompt patterns are optimized for high-velocity AI development — "vibe coding" sessions where agents generate, iterate, and ship at maximum speed.

### Pattern 1: Game UI Sprint

```
Build a health bar component with these constraints:
- WCAG 3.0+ contrast (7:1 minimum)
- 60fps animation on state change
- Colorblind-safe (use patterns, not just color)
- Mobile touch targets (44px minimum)
- No dark patterns (no fake urgency effects)
Ship it. Follow the Four Laws.
```

### Pattern 2: Rapid Prototype

```
Scaffold a settings menu with:
- Keyboard navigation (Tab/Arrow/Enter/Escape)
- Screen reader announcements on state change
- Persistent user preferences (localStorage with fallback)
- Responsive: mobile-first, desktop-enhanced
Use existing component patterns. Don't reinvent.
```

### Pattern 3: Iterative Refinement

```
The modal component works but needs:
1. Focus trap (Tab cycles within modal)
2. Escape key closes
3. Return focus to trigger on close
4. aria-modal="true" and role="dialog"
Read the current code first. Make minimal changes.
```

### Pattern 4: Full-Stack Feature

```
Add a leaderboard feature:
- Backend: REST endpoint, paginated, cached
- Frontend: Accessible table with sort controls
- Ethics: No addictive refresh patterns, show last-updated timestamp
- Performance: < 200ms response, skeleton loading state
Follow guardrails. Halt if auth model is unclear.
```

### Anti-Patterns to Avoid

| Don't | Do Instead |
|-------|------------|
| "Make it look good" | "Follow 2026_UI_UX_STANDARD.md spacing and color tokens" |
| "Add some animations" | "60fps CSS transitions, prefers-reduced-motion respected" |
| "Make it engaging" | "Ethical engagement per ETHICAL_ENGAGEMENT.md, no dark patterns" |
| "Just make it work" | "Implement with tests, accessibility, and error states" |
