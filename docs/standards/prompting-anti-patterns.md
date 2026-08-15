# Anti-Patterns and Troubleshooting

> What to avoid when prompting, and how to fix common problems

**See also:** [Prompting Guide overview](prompting-guide.md)

---

## Anti-Patterns to Avoid

### ❌ Anti-Pattern 1: Vague Requests

```
Make it better
```

**Problem:** AI doesn't know what "better" means.

**Fix:** Be specific about what "better" looks like.

### ❌ Anti-Pattern 2: Scope Creep

```
Fix the login bug, oh and also refactor the auth system, 
and update the docs, and add tests, and maybe redesign the UI
```

**Problem:** Too many unrelated tasks in one prompt.

**Fix:** One task per prompt, or clearly separate with "AFTER THIS, we'll do X"

### ❌ Anti-Pattern 3: Assumption of Knowledge

```
Fix the auth issue
```

**Problem:** AI doesn't know which auth issue unless you tell it.

**Fix:** Provide error messages, file names, reproduction steps.

### ❌ Anti-Pattern 4: Negative Constraints Only

```
Don't break anything
```

**Problem:** AI doesn't know what "anything" means.

**Fix:** Be explicit about what to preserve: "Maintain all existing tests" "Don't change public APIs"

### ❌ Anti-Pattern 5: Missing Context

```
Add the feature
```

**Problem:** No context about what the feature should do.

**Fix:** Describe the feature, provide user stories, show examples.

---

## Troubleshooting

### "AI keeps asking me questions"

**Cause:** Not enough context provided.

**Fix:** Add more detail about what you want, include examples.

### "AI is changing files I didn't ask for"

**Cause:** Scope not clearly defined.

**Fix:** Use "SCOPE - ONLY THESE FILES:" format.

### "AI is doing things in the wrong order"

**Cause:** Steps not explicitly sequenced.

**Fix:** Number the steps: "Step 1... Step 2... Step 3..."

### "AI is ignoring my constraints"

**Cause:** Constraints buried in text.

**Fix:** Use formatting:
```
CONSTRAINTS:
- Must do X
- Must not do Y
- Must use Z pattern
```

### "AI is over-engineering"

**Cause:** Requirements too open-ended.

**Fix:** Add constraints: "Keep it simple" "Use existing patterns" "Minimal changes"

### "AI is missing edge cases"

**Cause:** Edge cases not mentioned.

**Fix:** Explicitly list edge cases: "Handle empty input" "Handle network timeout" "Handle concurrent access"
