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

### 2026-03-22 - Transcript: large JSONL lines cause silent parse truncation

- **Issue**: Agents/tools not detected when JSONL transcript contains lines >1MB (e.g., tool_result with large file content)
- **Root Cause**: `bufio.Scanner` buffer max was 1MB; oversized lines cause `Scan()` to return false, silently stopping all further parsing
- **Solution**: Increased max buffer from 1MB to 16MB (initial buffer kept small at 64KB for memory efficiency)
- **Prevention**: Monitor for similar buffer-size assumptions in any streaming/line-based parser

### 2026-03-22 - Usage: empty parentheses when reset time unavailable

- **Issue**: Status line shows `()` when usage reset time has passed or is unavailable
- **Root Cause**: Format string `({5h_reset})` outputs parentheses even when var is empty
- **Solution**: Moved parentheses into the variable itself; empty reset time produces empty string

### 2026-10-04 - Tool results reset the per-turn tool state

- **Issue**: Completed/error highlighting only reflected the latest tool result batch (an earlier red ✗ disappeared after the next result). With tail scan active (transcript > `max_tail_size`), a still-running parallel tool was discarded as an orphan once a sibling tool finished.
- **Root Cause**: Turn boundary was `type == "user"`, but Claude Code records `tool_result` blocks in `"user"` entries too, so every tool result started a "new turn" and moved `lastUserTS` forward.
- **Solution**: `isUserPrompt` treats only non-`isMeta` user entries without `tool_result` blocks as turn boundaries.
- **Prevention**: Tests use realistic transcript shapes (tool results inside `"user"` entries).

### 2026-10-04 - Usage `bar_width` ignored, backoff checked after keychain read

- **Issue**: `usage.bar_width` was documented but the bars were always 10 wide. When the 429 backoff gate was reached (no last good data, or `cache_ttl_seconds` shorter than the backoff), each refresh still spawned `/usr/bin/security`, and a keychain failure there overwrote the cache and cleared `RetryAfterUntil`.
- **Root Cause**: Hardcoded `defaultBarSize`; backoff check came after `keychain.Read()`.
- **Solution**: `NewUsageModule` takes the bar width; backoff is checked before the lock and the credential read.
