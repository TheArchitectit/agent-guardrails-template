# Prompting Fundamentals

> The golden rules of writing prompts that work with Agent Guardrails

**See also:** [Prompting Guide overview](prompting-guide.md)

---

## The Golden Rules

### Rule 1: Start with Context

❌ **Bad:**
```
Fix the bug
```

✅ **Good:**
```
There's a bug in the authentication system where users can't log in with valid credentials. 

Context:
- Repository: myapp/backend
- File: src/auth/login.js
- Error: "Invalid credentials" even with correct password
- Database: PostgreSQL
- Framework: Express.js

Task: Find and fix the login bug. The issue is likely in the password comparison logic.
```

### Rule 2: Define Scope Explicitly

❌ **Bad:**
```
Update the API
```

✅ **Good:**
```
Update the user API endpoints to add email validation.

Scope:
- File: src/routes/users.js
- Only modify POST /api/users and PUT /api/users/:id
- Do NOT touch authentication or other routes
- Add validation using Joi schema
- Return 400 if email is invalid
```

### Rule 3: Provide Constraints

❌ **Bad:**
```
Refactor the code
```

✅ **Good:**
```
Refactor the data processing module to improve readability.

Constraints:
- Keep all existing functionality
- Maintain backward compatibility
- Don't change function signatures
- Add unit tests for new helper functions
- Use existing patterns from src/utils/helpers.js
```

### Rule 4: Include Examples

❌ **Bad:**
```
Add error handling
```

✅ **Good:**
```
Add error handling to the file upload endpoint.

Current code (src/routes/upload.js):
```javascript
app.post('/upload', (req, res) => {
  const file = req.files.file;
  fs.writeFileSync('/uploads/' + file.name, file.data);
  res.json({ success: true });
});
```

Expected behavior:
- Handle missing file: return 400 with error "No file provided"
- Handle file too large (>10MB): return 413 with error "File too large"
- Handle disk full: return 500 with error "Storage error"
- Always return JSON: { success: boolean, error?: string }

Example error response:
```json
{ "success": false, "error": "No file provided" }
```
```

> **Why explicit context enables speed:** Every detail you provide upfront is a clarification your AI agent doesn't need to ask for. Explicit context eliminates round-trips, reducing a 5-prompt conversation to a single generation. The most productive vibe coding sessions start with the richest prompts.

---

## Quick Reference Card

### Do ✅
- Provide context
- Define scope
- Give examples
- List constraints
- Specify format
- Include error cases
- Reference existing code

### Don't ❌
- Be vague
- Assume knowledge
- Skip error handling
- Ignore scope
- Rush to code
- Forget tests
- Break patterns

### Formatting Tips
- Use headers (##)
- Use lists (-)
- Use code blocks (```)
- Use bold for emphasis (**)
- Use emojis sparingly (✅ ❌)

### Keywords That Help
- "ONLY these files"
- "Follow this pattern"
- "Do NOT touch"
- "MUST do"
- "Step 1, Step 2"
- "For example"

---

## Practice Exercise

Try rewriting this bad prompt:

```
Fix the thing
```

Into a good prompt using what you learned:

<details>
<summary>Click to see example answer</summary>

```markdown
Task: Fix the memory leak in the data processing worker

PROBLEM:
The worker process memory grows indefinitely when processing large datasets.
After ~1000 records, memory usage exceeds 2GB and the process is killed.

CURRENT CODE (src/workers/dataProcessor.js):
```javascript
async function processBatch(records) {
  for (const record of records) {
    const result = await transform(record);
    await save(result);
  }
}
```

SCOPE:
- ONLY modify src/workers/dataProcessor.js
- May create helper functions in same file
- Do NOT change the database layer
- Do NOT modify the transform function

CONSTRAINTS:
- Memory usage must stay under 500MB for 10,000 records
- Maintain current throughput (1000 records/second)
- Don't break existing tests

ACCEPTANCE CRITERIA:
- [ ] Process 10,000 records with <500MB memory
- [ ] All existing tests pass
- [ ] No memory growth over time
- [ ] Code reviewed and approved

REFERENCES:
- Similar batch processing: src/utils/batchProcessor.js
```

</details>
