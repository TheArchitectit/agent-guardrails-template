# Social media story kit
Prepared Oct 1, 2026. DRAFTS FOR ROGER'S REVIEW. Nothing is posted. Do not post before Roger approves final wording and channels.

## Facts you may state (verified live Oct 1, 2026)
- Repo: TheArchitectit/agent-guardrails-template, public.
- Main branch was reduced to a single retirement README. Tag v3.7.1 still holds the code (816 files).
- Plan: reboot as an MCP server plus extensions; Copilot first.
## Roger's account (state as his experience, not established cause)
- During a secret-cleanup effort involving OpenClaw, the project came close to being lost.
## Do not say
OpenClaw acted maliciously; all code or history was destroyed; any figure for hours, users, stars growth, certifications; that connecting the MCP server makes an agent safe. Do not post logs, hostnames, IPs, credential names, network layout or screenshots of terminals.

## Core message (one line)
"Safety can't depend on an agent remembering to be careful."

## Long-form post (blog / LinkedIn / dev.to), ~350 words
**We almost lost Agent Guardrails. Here is what that taught us.**

We built Agent Guardrails because AI can move fast, and speed without boundaries can turn a small mistake into a big one.

While cleaning up secrets in our repository with an AI tool, we came close to losing work we had put real time and care into. We are still checking the exact sequence, so we won't guess at it here. What we can say for certain: the main branch ended up reduced to a retirement notice, and one release tag, v3.7.1, still held the code.

That was the lesson in miniature. We build safety tooling, and we were nearly caught by the problem it exists to prevent. Not because anyone meant harm. Because powerful tools, a cleanup goal and a missing boundary are enough.

So we are rebooting. Agent Guardrails becomes an MCP server with thin extensions for the editors people already use. Copilot first. Others as we test them.

Three principles come with it:

1. Boundaries over vigilance. Don't rely on an agent being careful. Give it limits and a way back.
2. Say what is enforced. Connecting an MCP server gives an agent access to checks. It does not force the agent to use them. We label every path as advisory, checked, or blocking, and we show "not enforced" when that's the truth.
3. Unknown is not a pass. If a check can't run, you see UNKNOWN, not a green light.

We're building this in the open, starting from the tag that survived. Come check our work, break it, and tell us what we got wrong.

[repo link]

## Short posts
**X/Bluesky/Threads single (under 280):**
We nearly lost our AI-safety project during an AI-assisted secret cleanup. The code survived on one tag. So we're rebooting Agent Guardrails as an MCP server + editor extensions, and we'll label what's enforced vs advisory. Open repo: [link]

**Thread (7 posts)**
1. We build guardrails for AI agents. Last week we nearly lost the project to the exact kind of mistake they prevent. Thread.
2. During an AI-assisted secret cleanup, the main branch ended up reduced to a retirement note. Tag v3.7.1 still held the code. We're still checking exactly how.
3. No villain here. A broad goal, a powerful tool, and no hard boundary was enough.
4. So: reboot. Agent Guardrails as an MCP server with thin extensions. Copilot in VS Code first.
5. Honesty rule: an MCP connection gives an agent access to checks; it doesn't force it to use them. We label each path advisory, checked or blocking.
6. Second rule: unknown is not a pass. If a check can't run, you see UNKNOWN.
7. We're building in the open from the surviving tag. Star it, break it, tell us what's wrong: [link]

**LinkedIn short:** Same as long-form paragraphs 1-2 plus principles; add a question: "What's your tested way back when an agent touches something it shouldn't?"

**Reddit/HN style (title only, post body = long form, follow each community's self-promo rules; read them first):**
- "We nearly lost our AI-safety repo to an AI-assisted cleanup. Here's the reboot plan."
- Avoid hype words. Answer every comment, including criticism. Do not ask for upvotes or stars.

## Video (75-90s) outline
See Plan B script. Hook: "Safety can't depend on an agent remembering to be careful." Beats: story (15s), the lesson (15s), demo of a blocked deletion on sanitized sample repo (25s), honesty table (15s), call to action (10s). Captions burned in. Own voice or text only; no voice cloning. Paid video services need Roger's word.

## Content calendar (first 30 days after Roger approves)
- Day 0: repo story + README update, long-form post, thread.
- Day 3: short demo clip (blocked deletion).
- Day 7: "What 'advisory' means" explainer post.
- Day 10: quickstart video.
- Day 14: first monthly safety note, including what failed.
- Day 21: good-first-issues call, contributor thank-yous.
- Day 28: Copilot setup walkthrough once B02 passes.

## Engagement rules
- Reply honestly; concede valid criticism publicly and open an issue.
- Never claim more than the evidence page shows.
- No bought engagement, no follow-for-follow rings, no sock accounts.
- Credit people who file issues or PRs.

## Review checklist before posting
[ ] Roger approved text  [ ] scrub scan clean  [ ] links work  [ ] claims match matrix  [ ] OpenClaw wording confirmed  [ ] captions present

