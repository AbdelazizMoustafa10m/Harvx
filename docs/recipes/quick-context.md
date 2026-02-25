# Recipe: Quick Context (Alex Persona)

Quick commands for getting project context into an LLM conversation.

## Use Case

You want to quickly give an LLM context about your project for a chat session—asking questions, getting explanations, or having quick coding discussions.

## Recipe 1: Full Project Brief

```bash
# Generate a project brief and copy to clipboard (macOS)
harvx brief --stdout | pbcopy

# Generate a project brief and copy to clipboard (Linux)
harvx brief --stdout | xclip -selection clipboard
```

## Recipe 2: Claude-Optimized Brief

```bash
# XML format optimized for Claude
harvx brief --target claude --stdout | pbcopy
```

## Recipe 3: Module-Specific Context

```bash
# Get context about a specific module
harvx slice --path internal/auth --stdout | pbcopy

# Multiple modules at once
harvx slice --path internal/auth --path internal/middleware --stdout | pbcopy
```

## Recipe 4: Automatic Session Bootstrap

Set up once and every Claude Code session starts with context:

1. Create `.claude/hooks.json` in your project:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "command": "harvx brief --stdout",
        "timeout": 5000
      }
    ]
  }
}
```

2. Start a new Claude Code session—context is injected automatically.

## Recipe 5: Full Repository Context

```bash
# Generate complete repository context for a long conversation
harvx generate --target claude --stdout | pbcopy
```

Use this for in-depth discussions where you need the agent to understand the full codebase.

## Recipe 6: Check What's Included

```bash
# Preview what would be included (no generation)
harvx preview

# See token budget heatmap
harvx preview --heatmap

# Get machine-readable preview
harvx preview --json
```

## Tips

- Use `--target claude` when pasting into Claude conversations for XML formatting
- Use `--quiet` to suppress diagnostic messages when piping output
- The brief is typically 1-4K tokens—small enough for any context window
- Use `--max-tokens` to control budget: `harvx brief --max-tokens 2000 --stdout`
