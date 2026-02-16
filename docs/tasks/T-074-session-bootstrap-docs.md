# T-074: Session Bootstrap Documentation and Claude Code Hooks Integration

**Priority:** Must Have
**Effort:** Medium (6-10hrs)
**Dependencies:** T-070 (Brief Command), T-072 (Slice Command), T-073 (Workspace Command)
**Phase:** 5 - Workflows

---

## Description

Create comprehensive documentation for integrating Harvx with coding agent session bootstrap workflows. This includes a reference Claude Code `SessionStart` hook configuration, a lean `CLAUDE.md` template, documentation for `--target claude` rendering mode, and usage guides for composing `brief`, `slice`, and `workspace` commands in session startup scripts. The goal is that developers can copy-paste the documented setup and immediately get project-aware agent sessions.

## User Story

As a Claude Code user, I want clear, copy-paste-ready documentation for setting up Harvx as a session bootstrap tool so that every new session starts with project awareness without me maintaining a bloated CLAUDE.md.

## Acceptance Criteria

- [ ] `docs/guides/session-bootstrap.md` contains:
  - Overview of the session bootstrap strategy (lean CLAUDE.md + dynamic Harvx context)
  - Reference Claude Code `SessionStart` hook configuration:
    ```json
    {
      "hooks": {
        "SessionStart": [
          {
            "command": "harvx brief --profile session --stdout",
            "timeout": 5000
          }
        ]
      }
    }
    ```
  - Hook placement instructions (`.claude/hooks.json` in project root or `~/.claude/hooks.json` global)
  - Explanation of what the hook outputs and how Claude uses it
  - Performance guidance: brief should complete in under 2 seconds for session startup
  - Troubleshooting: common issues (harvx not in PATH, timeout too short, profile not found)
- [ ] `docs/templates/CLAUDE.md` provides a lean baseline template:
  - Project principles and coding conventions (rules only, not architecture dumps)
  - Reference to Harvx for dynamic context: "Run `harvx brief` for project overview"
  - Kept under 500 tokens to avoid instruction bloat
- [ ] `--target claude` rendering mode is documented:
  - Outputs XML format following Anthropic's recommended tag structure
  - Sets default budget to 200K tokens
  - XML structure uses `<document>`, `<source>`, `<content>` tags
- [ ] `docs/guides/review-pipeline.md` contains:
  - End-to-end pipeline setup guide
  - Shell script example combining `brief` + `review-slice`
  - CI integration example (GitHub Actions workflow)
  - Canonical recipes for the three personas (Alex, Zizo, Jordan)
- [ ] `harvx slice --path <module>` is documented as the on-demand context tool for agents
- [ ] Workspace integration for multi-repo session bootstrap is documented
- [ ] All documentation includes working command examples that can be copy-pasted

## Technical Notes

- Claude Code hooks documentation: hooks fire at session start and inject the command's stdout as context into the conversation
- The hook timeout of 5000ms gives plenty of headroom for `harvx brief` (target: <2s)
- The `SessionStart` hook receives input data including `session_id`, `cwd`, `model`; output can include `additionalContext`
- Lean CLAUDE.md philosophy: rules and conventions only, delegate architecture/structure knowledge to Harvx
- For `--target claude`, the XML format should follow Anthropic's best practices for XML tags in prompts
- The review pipeline guide should show the complete flow: brief -> review-slice -> combine with diff -> feed to agents
- Include a `.github/workflows/harvx-review.yml` example for GitHub Actions integration
- Document the `HARVX_` environment variables that are useful in CI contexts
- Reference: PRD Sections 5.11.2 (Session Bootstrap), 5.7 (target presets), Anthropic XML tag docs

## Files to Create/Modify

- `docs/guides/session-bootstrap.md` - Session bootstrap documentation
- `docs/guides/review-pipeline.md` - Review pipeline integration guide
- `docs/templates/CLAUDE.md` - Lean CLAUDE.md template
- `docs/templates/hooks.json` - Reference Claude Code hooks configuration
- `docs/guides/workspace-setup.md` - Multi-repo workspace documentation
- `docs/recipes/` - Directory for persona-specific recipe files
- `docs/recipes/quick-context.md` - Alex persona (quick chat use)
- `docs/recipes/pipeline-review.md` - Zizo persona (pipeline integration)
- `docs/recipes/ci-integration.md` - Jordan persona (CI/CD setup)

## Testing Requirements

- Verify all code examples in documentation are syntactically valid
- Verify the hooks.json template is valid JSON
- Verify the CLAUDE.md template is under 500 tokens (use tiktoken to count)
- Verify all `harvx` commands referenced in docs are registered subcommands
- Review documentation for completeness: a developer unfamiliar with Harvx should be able to set up session bootstrap by following the guide
- Verify GitHub Actions workflow example uses valid syntax