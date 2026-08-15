# Common Prompt Patterns

> Reusable patterns for structuring prompts that produce reliable results

**See also:** [Prompting Guide overview](prompting-guide.md)

---

## Pattern 1: The Scoped Request

Use this when you want to limit what the AI touches.

```markdown
Task: Add input validation to the login form

SCOPE - ONLY THESE FILES:
- src/components/LoginForm.jsx
- src/validation/auth.js (create if doesn't exist)

DO NOT TOUCH:
- Authentication logic
- Backend API
- Other components

Validation rules:
- Email must be valid format
- Password must be 8+ characters
- Show inline errors below each field
```

## Pattern 2: The Step-by-Step

Use this for complex tasks that need to be broken down.

```markdown
Task: Implement user profile page

Step 1: Create the basic component structure
- Create src/pages/Profile.jsx
- Add route in App.jsx
- Create basic layout with sections

Step 2: Add data fetching
- Fetch user data from /api/user
- Handle loading state
- Handle error state

Step 3: Add edit functionality
- Make fields editable
- Add save/cancel buttons
- Implement update API call

Step 4: Testing
- Test with different user types
- Verify error handling
- Check responsive design

PAUSE after each step and ask for confirmation before proceeding.
```

## Pattern 3: The Reference Pattern

Use this when you want the AI to follow existing patterns.

```markdown
Task: Create a new API endpoint for user preferences

Follow the exact same pattern as src/routes/users.js:
- Use the same middleware structure
- Same error handling approach
- Same response format
- Same authentication checks

Specific requirements:
- GET /api/users/:id/preferences
- PUT /api/users/:id/preferences
- Validate input using Joi (like in users.js)
- Return 404 if user not found
```

## Pattern 4: The Validation Gate

Use this when you want checkpoints.

```markdown
Task: Refactor the database layer

Before making ANY changes:
1. Read and summarize the current implementation
2. Identify all files that will be affected
3. List potential risks
4. Propose a rollback strategy

After I approve:
5. Make the changes
6. Run tests
7. Verify no regressions

Do NOT proceed past step 4 without my explicit approval.
```

## Pattern 5: The Context-Rich

Use this when the task needs lots of background.

```markdown
Task: Fix the caching issue in the product catalog

BACKGROUND:
We're experiencing cache stampede during flash sales. When a popular product's cache expires, multiple requests hit the database simultaneously, causing slowdowns.

CURRENT IMPLEMENTATION:
- File: src/services/cache.js
- Uses Redis with 5-minute TTL
- No locking mechanism
- Cache key: product:${id}

PROPOSED SOLUTION:
Implement cache warming with stale-while-revalidate pattern:
1. Serve stale data while refreshing in background
2. Add probabilistic early expiration
3. Implement request coalescing

REFERENCES:
- Similar implementation: src/services/userCache.js
- Redis docs: https://redis.io/docs/manual/patterns/

ACCEPTANCE:
- Load test shows <100ms response time during cache miss
- No database connection spikes
- Graceful degradation when Redis is down
```
