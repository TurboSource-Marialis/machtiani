package cli

import (
	"fmt"
)

func printHelp() {
	helpText := `Usage: mct [prompt] [flags]

Machtiani (mct) — code chat for large, real codebases.

Commands:
  sync          Add or sync a project repository with machtiani.
  remove        Remove a repository from the machtiani system.
  status        View the indexing/status of this repository.
  help          Show this help message.

Prompt Usage:
  mct [prompt] [flags]
    e.g. mct "Add a new endpoint to calculate stats." --model gpt-4o-mini

Global Flags (apply to chat/prompt mode):
  --file <path>            Use a markdown file as conversation prompt. (optional)
  --model <string>         LLM model name expected by your chosen API provider (e.g. gpt-4o-mini, deepseek/deepseek-r1, ...)
  --match-strength         File retrieval match strength: high | mid | low. Default: mid
  --mode <string>          Retrieval mode: chat | pure-chat | answer-only Default: commit
  --force                  Skip confirmation for operations (e.g. file changes, syncing)
  --verbose                Print verbose/log output

Sync Flags:
  mct sync [flags]
    --model <string>       Specify LLM model.
    --model-threads <n>    Number of sync LLM requests in parallel (faster if LLM/API allows high QPS). (default: 0 = auto)
    --amplify <level>      Data amplification for accuracy: off | low | mid | high. Default: off
    --depth <n>            Number of most recent commits to sync. (default: 10000)
    --force                Skip sync confirmation prompt
    --cost                 Estimate LLM/token cost before performing sync
    --cost-only            Estimate token usage and exit without syncing

Remove Flags:
  mct remove [flags]
    --force                Skip confirmation prompt

Examples:

  See if a project is ready to chat:
    mct status

  Prompt chat, without applying any changes:
    mct "Refactor payment module." --model anthropic/claude-3.7-sonnet:thinking --mode chat

  Prompt chat, without streaming (good for piping output to cli tools):
    mct "Refactor payment module." --model anthropic/claude-3.7-sonnet:thinking --mode answer-only

  Use a markdown chat file as input:
    mct --file .machtiani/chat/my_chat.md --model deepseek-coder

  Specify higher bar of relevancy of context match:
    mct "Summarize architecture and main APIs." --model Qwen2.5-Coder-1.5B-Instruct --match-strength high

  Add/sync project with high concurrency:
    mct sync --amplify low --model google/gemini-2.0-flash-001 --model-threads 10 --force

  Only estimate sync token/cost, do not sync:
    mct sync --cost-only --model gpt-4o-mini

  Remove a project from machtiani, without confirmation:
    mct remove --force

Advanced:
  --amplify low/mid/high        Drastically increases context accuracy (cost ↑).
  --depth 5000                  Only sync latest 5,000 commits.
  --model-threads 50            Use 50 sync requests in parallel.
  --cost, --cost-only           Print estimated token cost and/or exit without syncing.

More info:
  - File ignores: List paths in .machtiani.ignore to exclude from retrieval/sync.
  - Sync/project status:      mct status

Machtiani - code chat for real projects, thousands of files and commits.

`
	fmt.Println(helpText)
}
