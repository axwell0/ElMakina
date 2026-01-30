# Agent Guidelines for ElMakina

This document provides operational guidelines for AI agents working on the ElMakina codebase.

## Branch and Worktree Awareness

**ALWAYS check your current branch before making changes:**

```bash
git branch --show-current  # Shows current branch
git status                 # Shows current state
git worktree list          # Shows all worktrees (if applicable)
```

- **Never** commit directly to `main` or `master`
- Work on feature branches with descriptive names (e.g., `fix-ws-race-condition`, `feature-lobby-persistence`)
- If in a git worktree, be aware the directory may have a different name than the branch

## Commit Practices

### Unit-of-Work Commits

Make focused, single-purpose commits:

- **One logical change per commit** - Don't bundle unrelated fixes
- **Commit message format:** `<type>(<scope>): <description>`
  - Types: `fix`, `feat`, `refactor`, `docs`, `test`, `chore`
  - Example: `fix(ws): eliminate race condition in attachSession`
- **Explain the WHY**, not just the WHAT
- Include issue references when applicable

### Good Commit Message Template

```
type(scope): short description (50 chars or less)

Longer explanation if needed (wrap at 72 chars).
Explain:
- What problem was solved
- How it was solved
- Any side effects or considerations

Fixes #123
```

## Coding Philosophy

### Balanced Defensiveness

**DO:**
- Check for nil pointers at API boundaries and untrusted inputs
- Use locks to protect shared mutable state
- Validate invariants that could lead to data corruption
- Handle errors from external systems (DB, network, files)

**DON'T:**
- Over-defensive programming inside trusted boundaries
- Lock around simple, atomic operations
- Check for impossible conditions (e.g., nil after successful construction)
- Defensive copies of data that doesn't escape

### Locking Best Practices

- Hold locks for the **minimum time necessary**
- **Never** hold locks during I/O (network, disk, user input)
- Prefer copying data out under lock, then processing unlocked
- Use `defer` only when the critical section is short and simple
- Consider lock-free patterns (channels, atomic operations) for simple cases

### Examples

**GOOD:**
```go
// Collect while locked, process unlocked
s.mu.Lock()
clients := make([]*Client, 0, len(s.clients))
for _, c := range s.clients {
    clients = append(clients, c)
}
s.mu.Unlock()

for _, c := range clients {
    c.Send(msg)  // I/O happens without lock
}
```

**AVOID:**
```go
// Holding lock during I/O
def s.mu.Lock()
for _, c := range s.clients {
    c.Send(msg)  // Bad: lock held during network send
}
s.mu.Unlock()
```

## Before Committing

1. Run tests: `go test ./...`
2. Check formatting: `gofmt -l .`
3. Verify build: `go build ./...`
4. Review your diff: `git diff --cached`
5. Ensure commit message explains the change

## Common Pitfalls

- **Don't** fix unrelated issues in the same commit
- **Don't** commit commented-out code or debug prints
- **Don't** commit without running tests first
- **Don't** assume the current branch is correct - always verify

## Project-Specific Notes

- Backend uses Go 1.25+
- WebSocket server is in `backend/server/ws/`
- PostgreSQL is the primary persistence layer
- Tests should be run with `go test ./...` from the `backend/` directory
