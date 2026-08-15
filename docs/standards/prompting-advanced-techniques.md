# Advanced Prompting Techniques

> Techniques for complex or nuanced prompting scenarios

**See also:** [Prompting Guide overview](prompting-guide.md)

---

## Technique 1: Progressive Disclosure

Start simple, add complexity only if needed.

```markdown
Initial Task: Create a simple user registration form

If validation passes, also:
- Add email verification
- Implement rate limiting
- Add CAPTCHA for suspicious IPs

But ONLY do the extras if the basic form works perfectly.
```

## Technique 2: Constraint Programming

Define what NOT to do explicitly.

```markdown
Task: Optimize the search query

CONSTRAINTS - NEVER DO:
- Don't use raw SQL (use ORM)
- Don't remove existing indexes
- Don't change the API response format
- Don't break pagination
- Don't ignore security (always use parameterized queries)

MUST DO:
- Add database query logging
- Keep response time under 200ms
- Handle empty results gracefully
- Maintain backward compatibility
```

## Technique 3: Example-Driven

Show exactly what you want.

```markdown
Task: Add a new component for user cards

Here's the EXACT pattern to follow (from src/components/ProductCard.jsx):

```jsx
const ProductCard = ({ product }) => {
  return (
    <Card>
      <Card.Header>
        <h3>{product.name}</h3>
      </Card.Header>
      <Card.Body>
        <p>{product.description}</p>
        <Badge>{product.category}</Badge>
      </Card.Body>
    </Card>
  );
};
```

Now create UserCard following this EXACT same structure, just with user data instead of product data.
```

## Technique 4: Hypothetical Reasoning

Ask the AI to think through scenarios.

```markdown
Task: Implement a payment retry mechanism

Before coding, walk through these scenarios:

Scenario 1: Network timeout
- What should happen?
- How many retries?
- What's the backoff strategy?

Scenario 2: Insufficient funds
- Should we retry?
- What error message?

Scenario 3: Duplicate payment attempt
- How do we detect it?
- How do we prevent it?

After analyzing, implement the solution that handles all three.
```

## Technique 5: Role Play

Set a specific persona for better results.

```markdown
You are a senior security engineer with 10 years of experience.

Task: Review this authentication code for security vulnerabilities.

Approach:
- Think like an attacker
- Look for OWASP Top 10 issues
- Consider edge cases
- Question every assumption

Code to review:
[code here]

Provide:
1. List of vulnerabilities found
2. Severity rating for each
3. Suggested fixes with code examples
4. Any additional security recommendations
```
