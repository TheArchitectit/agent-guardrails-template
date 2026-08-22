# Atlas Cloud (Open Source Sponsorship)

Atlas Cloud for Open Source sponsors **Agent Guardrails Template** — a **Starter** grant of
**$50/month** in credits, usable across Atlas Cloud's 300+ image, video, audio, 3D, and LLM
models. This page documents how to route guardrails' AI-backed checks through Atlas instead
of a pay-per-use OpenAI/Claude endpoint, and how to keep the grant active.

> Because this is an open-source sponsorship, it is **community-facing documentation** — no
> private API key or personal credentials are included here. Set your own `ATLAS_API_KEY`
> in `.env` (gitignored).

## Why Atlas

Guardrails verifies content safety, runs AI advisors, and can gate AI output through a
model-backed validator. Those check calls normally cost money per request. Atlas Cloud
provides an OpenAI-compatible API to a broad model catalog, so you can run the same checks
on sponsored credits instead of your own bill.

## Config

Add to `.env` (copy from `.env.example`):

```bash
# Atlas Cloud (sponsored compute) — OpenAI-compatible provider
ATLAS_BASE_URL=https://api.atlascloud.ai/v1
ATLAS_API_KEY=your_atlas_api_key
ATLAS_MODEL=               # any Atlas-hosted LLM supporting OpenAI chat completions
LLM_PROVIDER=openai        # openai | claude | atlas
```

Set `LLM_PROVIDER=atlas` to route content-safety and AI-advisor checks through Atlas.

## Keeping the grant active

Atlas Cloud pauses the grant if either happens for two consecutive months:

1. **No genuine API usage.** Run real guardrails check calls through Atlas — the integration
   exists exactly so sponsored credits get used.
2. **Removed sponsorship badge.** Keep the **Powered by Atlas Cloud** badge in the README
   (it carries the project's referral link). The badge is already present on the repo README.

## Verification

```bash
curl -s "$ATLAS_BASE_URL/models" \
  -H "Authorization: Bearer $ATLAS_API_KEY" | head
```

And a chat-completions smoke test through the configured `ATLAS_MODEL`. If both succeed,
guardrails' model-backed checks will run on Atlas credits.
