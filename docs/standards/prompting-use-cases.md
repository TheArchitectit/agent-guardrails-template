# Prompt Examples by Use Case

> Worked examples for API development, frontend components, database migrations, and DevOps

**See also:** [Prompting Guide overview](prompting-guide.md)

---

## Use Case 1: API Development

```markdown
Task: Create REST API endpoints for a blog

SCOPE:
- Base path: /api/v1/posts
- Files: src/routes/posts.js (new)

ENDPOINTS:

GET /api/v1/posts
- Query params: page, limit, sort
- Returns: { posts: [], total: number, page: number }
- Pagination: default 20 items per page

GET /api/v1/posts/:id
- Returns: { post: { id, title, content, author, created_at } }
- 404 if not found

POST /api/v1/posts
- Body: { title: string (required), content: string (required) }
- Validation: title min 5 chars, content min 50 chars
- Returns: { post: { id, ... } }
- 400 if validation fails with error details

PUT /api/v1/posts/:id
- Body: partial update (only provided fields)
- Returns updated post
- 404 if not found

DELETE /api/v1/posts/:id
- Returns: 204 No Content
- 404 if not found

TECHNICAL:
- Use Express.js
- Use existing auth middleware from src/middleware/auth.js
- Use existing Post model from src/models/Post.js
- Follow error handling pattern from src/routes/users.js
- Add tests in tests/routes/posts.test.js
```

## Use Case 2: Frontend Component

```markdown
Task: Create a reusable Modal component

SPECIFICATIONS:

Props:
- isOpen: boolean (required)
- onClose: function (required)
- title: string
- children: ReactNode
- size: 'small' | 'medium' | 'large' (default: 'medium')
- closeOnOverlayClick: boolean (default: true)
- showCloseButton: boolean (default: true)

Behavior:
- Click outside modal closes it (if enabled)
- ESC key closes modal
- Focus trap inside modal
- Return focus to trigger element on close
- Animate in/out (fade + scale)

Accessibility:
- aria-modal="true"
- role="dialog"
- aria-labelledby pointing to title
- Focus management

Styling:
- Use Tailwind CSS
- Backdrop: bg-black/50
- Modal: bg-white rounded-lg shadow-xl
- Sizes:
  - small: max-w-md
  - medium: max-w-lg
  - large: max-w-2xl

Usage Example:
```jsx
<Modal
  isOpen={showModal}
  onClose={() => setShowModal(false)}
  title="Confirm Delete"
  size="small"
>
  <p>Are you sure?</p>
  <Button onClick={handleDelete}>Delete</Button>
</Modal>
```

Files:
- Create: src/components/Modal.jsx
- Create: src/components/Modal.test.jsx
```

## Use Case 3: Database Migration

```markdown
Task: Add user preferences table

CURRENT STATE:
Users table has: id, email, password_hash, created_at

MIGRATION:
- Create user_preferences table
- Columns:
  - id: UUID, primary key
  - user_id: UUID, foreign key to users.id, onDelete CASCADE
  - theme: ENUM('light', 'dark', 'system'), default 'system'
  - notifications_enabled: BOOLEAN, default true
  - language: VARCHAR(10), default 'en'
  - created_at: TIMESTAMP
  - updated_at: TIMESTAMP

CONSTRAINTS:
- One preference row per user
- Auto-update updated_at on change

FILES:
- migration: migrations/20240215_add_user_preferences.sql
- model: src/models/UserPreferences.js
- relation: Update src/models/User.js to include hasOne

TESTING:
- Verify migration rolls forward
- Verify migration rolls back
- Test foreign key constraint
- Test default values

DO NOT:
- Modify existing users table
- Delete any data
- Break existing queries
```

## Use Case 4: DevOps/Infrastructure

```markdown
Task: Set up CI/CD pipeline for automated testing

CURRENT STATE:
- GitHub repository
- No CI/CD configured
- Tests exist: npm test
- Linting: npm run lint

REQUIREMENTS:

Pipeline Triggers:
- On every PR to main
- On every push to main

Jobs:

1. Lint:
   - Run: npm run lint
   - Fail on warnings

2. Test:
   - Run: npm test
   - Generate coverage report
   - Upload coverage to Codecov
   - Require 80% coverage

3. Build:
   - Run: npm run build
   - Cache node_modules
   - Upload build artifacts

4. Security Scan:
   - Run: npm audit
   - Fail on high/critical vulnerabilities

5. Deploy (main branch only):
   - Deploy to staging environment
   - Run smoke tests
   - If smoke tests pass, deploy to production

CONFIGURATION:
- File: .github/workflows/ci.yml
- Use GitHub Actions
- Use latest LTS Node.js
- Set timeout: 30 minutes

NOTIFICATIONS:
- Slack webhook on failure
- PR comments with test results
```
