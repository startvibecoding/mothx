// Package systeminit builds the prompt used by the /systeminit command to
// generate (or refresh) a project-level AGENTS.md guide for AI agents.
//
// The same prompt text is shared by the TUI, CLI, and ACP entry points so the
// behavior stays consistent across surfaces. In interactive surfaces (TUI/ACP)
// the agent is encouraged to use the `question` tool to clarify project
// conventions with the user before writing the file, yielding a higher quality
// AGENTS.md. In non-interactive surfaces (CLI print mode) the agent writes the
// file directly from what it can infer from the codebase.
package systeminit

import "strings"

// Command is the slash command that triggers system initialization.
const Command = "/systeminit"

// baseInstructions describes what a good AGENTS.md should contain. Shared by
// both the interactive and non-interactive prompts.
const baseInstructions = `You are setting up this project for future AI coding agents by creating a high-quality AGENTS.md file at the repository root.

First, investigate the project thoroughly:
- Detect the primary language(s), frameworks, and project layout.
- Identify build, test, run, and lint commands (look at Makefile, package.json scripts, go.mod, pyproject.toml, Cargo.toml, etc.).
- Note important directories and what they contain.
- Infer coding conventions, architecture patterns, and any existing rules (.editorconfig, linters, existing AGENTS.md/CLAUDE.md/.cursorrules).

Then write a concise, actionable AGENTS.md at the project root with sections such as:
- Project snapshot (language, frameworks, purpose)
- Important directories
- Architecture notes
- Build / test / run commands
- Coding conventions and working rules
- Anything an agent must NOT do

Keep it focused and practical. Prefer real, verified commands and paths over guesses. If an AGENTS.md already exists, improve it in place rather than discarding useful content.`

// interactiveInstructions adds guidance to ask the user clarifying questions
// before writing the file.
const interactiveInstructions = `

Before writing the file, use the ` + "`question`" + ` tool to ask the user a few (3-6) high-value clarifying questions about things you cannot reliably infer from the code alone, for example:
- The project's main purpose and target users
- Preferred build/test/release workflow and any commands to always or never run
- Coding conventions, formatting, or review expectations
- Constraints or guardrails agents must respect (files to avoid, no auto-commit, etc.)
- Priorities (performance, readability, backward compatibility, etc.)

Ask one focused question at a time with clear predefined options (the user can always type a custom answer). Skip questions whose answers are already obvious from the codebase. After the user answers, incorporate their guidance and write the final AGENTS.md.`

const finalNote = `

When done, briefly summarize what you wrote and where.`

// Prompt returns the system-init instruction prompt. When interactive is true
// the agent is told to use the question tool to clarify with the user first.
// extra carries optional user-provided guidance (e.g. "ask me in Chinese, write
// AGENTS.md in English") appended as additional instructions.
func Prompt(interactive bool, extra string) string {
	var b strings.Builder
	b.WriteString(baseInstructions)
	if interactive {
		b.WriteString(interactiveInstructions)
	}
	if strings.TrimSpace(extra) != "" {
		b.WriteString("\n\nAdditional user instructions (follow these closely):\n")
		b.WriteString(strings.TrimSpace(extra))
	}
	b.WriteString(finalNote)
	return b.String()
}
