# Prompt Templates

> Ready-to-use templates for common development tasks

**See also:** [Prompting Guide overview](prompting-guide.md)

---

## Template 1: Feature Implementation

```markdown
## Feature: [Feature Name]

### Context
[Background information about the feature]

### Requirements
- [ ] Requirement 1
- [ ] Requirement 2
- [ ] Requirement 3

### Scope
- Files to modify: [list files]
- Files to NOT touch: [list files]
- New files to create: [list files]

### Technical Details
- Framework: [framework]
- Language: [language]
- Patterns to follow: [reference existing code]

### Acceptance Criteria
1. [Criteria 1]
2. [Criteria 2]
3. [Criteria 3]

### Testing
- [ ] Unit tests written
- [ ] Integration tests pass
- [ ] Manual testing completed

### Additional Notes
[Any special considerations]
```

## Template 2: Bug Fix

```markdown
## Bug Fix: [Bug Title]

### Problem
[Clear description of the bug]

### Steps to Reproduce
1. Step 1
2. Step 2
3. Step 3

### Expected Behavior
[What should happen]

### Actual Behavior
[What actually happens]

### Context
- File(s) involved: [list]
- Error message: [if any]
- Environment: [dev/staging/prod]

### Root Cause (if known)
[Your analysis]

### Proposed Solution
[Your suggestion, or leave blank]

### Testing After Fix
- [ ] Reproduction steps no longer trigger bug
- [ ] Related functionality still works
- [ ] Edge cases handled
```

## Template 3: Code Review

```markdown
## Code Review Request

### PR/MR Information
- Branch: [branch name]
- Changes: [files modified]
- Lines changed: [+X, -Y]

### Focus Areas
- [ ] Logic correctness
- [ ] Edge cases
- [ ] Performance
- [ ] Security
- [ ] Style/consistency

### Specific Questions
1. [Question 1]
2. [Question 2]

### Skip These
- [ ] Nitpicks (formatting)
- [ ] Out of scope files
- [ ] Known issues

### Timeline
[Urgency level]
```

## Template 4: Refactoring

```markdown
## Refactoring: [Area]

### Current State
[What's wrong with current code]

### Target State
[What it should look like]

### Constraints
- [ ] No functionality changes
- [ ] All tests must pass
- [ ] Maintain backward compatibility
- [ ] Update documentation

### Files
- Primary: [main file(s)]
- Dependencies: [files that depend on these]
- Tests: [test files to update]

### Patterns to Follow
- [Reference to similar code]

### Success Criteria
- [ ] Code is cleaner/more readable
- [ ] All tests pass
- [ ] No regressions
```

## Template 5: Documentation

```markdown
## Documentation Task

### Type
- [ ] API docs
- [ ] User guide
- [ ] README update
- [ ] Architecture doc
- [ ] Inline comments

### Target Audience
[Who will read this]

### Content Outline
1. [Section 1]
2. [Section 2]
3. [Section 3]

### Reference Materials
- [Link 1]
- [Link 2]

### Style Guide
- [ ] Follow existing patterns
- [ ] Include code examples
- [ ] Add diagrams if helpful
- [ ] Keep under 500 lines per doc
```
