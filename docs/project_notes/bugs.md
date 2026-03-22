# Bug Log

## Format

- Date, brief description, root cause, solution, prevention notes

### 2026-03-21 - Todo sync: deleted tasks inflate count

- **Issue**: Deleted tasks still counted in total, showing incorrect progress (e.g., "3/8" when only 5 real tasks)
- **Root Cause**: `TaskUpdate(status=deleted)` not filtered from transcript parse results
- **Solution**: Added post-parse filter to remove todos with `deleted` status
- **Prevention**: See ADR-008

### 2026-03-21 - Todo sync: batch detection fails across TaskUpdate gaps

- **Issue**: New plan's tasks appended to old plan instead of replacing it
- **Root Cause**: `TaskUpdate` didn't reset `consecutiveCreates` counter, so the batch detection couldn't see the boundary between old and new create phases
- **Solution**: Reset `consecutiveCreates` on `TaskUpdate` (signals end of create phase)
- **Prevention**: See ADR-005 update

### 2026-03-21 - Todo sync: missing completion events

- **Issue**: Tasks stuck as `in_progress` in statusline even though Claude Code UI shows them complete
- **Root Cause**: Claude Code often skips explicit `TaskUpdate(completed)` events in transcript JSONL
- **Solution**: Auto-complete heuristic: when a new task enters `in_progress`, all previously `in_progress` tasks are auto-completed
- **Prevention**: See ADR-007

### 2026-03-22 - Usage: empty parentheses when reset time unavailable

- **Issue**: Status line shows `()` when usage reset time has passed or is unavailable
- **Root Cause**: Format string `({5h_reset})` outputs parentheses even when var is empty
- **Solution**: Moved parentheses into the variable itself; empty reset time produces empty string
