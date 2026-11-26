## ADDED Requirements

### Requirement: Template audit work item identifiers are unique
The template audit gap register SHALL assign a unique identifier to each open and archived work item. The integration audit, verification queue, and numbered change path SHALL refer to the same work item when they use that identifier.

#### Scenario: New audit proposals coexist with archived audit work
- **WHEN** an architecture proposal is added to the template audit register
- **THEN** its identifier is distinct from every existing register row, resolves to the same proposal across the integration audit and verification queue, and leaves prior archive identifiers attached to their original evidence.
