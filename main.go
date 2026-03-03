package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Persona struct {
	Name   string `json:"name"`
	Emoji  string `json:"emoji"`
	System string `json:"system"`
}

type Opinion struct {
	Persona Persona
	Content string
	Round   int
}

type Usage struct {
	InputTokens  int
	OutputTokens int
	Provider     string
}

type CostTracker struct {
	mu       sync.Mutex
	usages   []Usage
	provider string
	model    string
}

func (ct *CostTracker) Add(u Usage) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.usages = append(ct.usages, u)
}

func (ct *CostTracker) Total() (int, int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	var in, out int
	for _, u := range ct.usages {
		in += u.InputTokens
		out += u.OutputTokens
	}
	return in, out
}

func (ct *CostTracker) Cost() float64 {
	in, out := ct.Total()
	switch {
	case strings.Contains(ct.model, "haiku"):
		return float64(in)*0.25/1e6 + float64(out)*1.25/1e6
	case strings.Contains(ct.model, "sonnet"):
		return float64(in)*3.0/1e6 + float64(out)*15.0/1e6
	case strings.Contains(ct.model, "gpt-4o-mini"):
		return float64(in)*0.15/1e6 + float64(out)*0.60/1e6
	case strings.Contains(ct.model, "gpt-4o"):
		return float64(in)*2.5/1e6 + float64(out)*10.0/1e6
	default:
		return float64(in)*3.0/1e6 + float64(out)*15.0/1e6
	}
}

func (ct *CostTracker) Print() {
	in, out := ct.Total()
	cost := ct.Cost()
	fmt.Printf("\n📊 Token Usage\n")
	fmt.Println("─────────────────────────────────")
	fmt.Printf("   Provider:  %s\n", ct.provider)
	fmt.Printf("   Model:     %s\n", ct.model)
	fmt.Printf("   Input:     %d tokens\n", in)
	fmt.Printf("   Output:    %d tokens\n", out)
	fmt.Printf("   Total:     %d tokens\n", in+out)
	fmt.Printf("   Cost:      $%.4f\n", cost)
}

var defaultTeam = []Persona{
	{
		Name:  "Developer",
		Emoji: "💻",
		System: `You are a blunt senior backend developer. 10 years experience. You hate unnecessary complexity.
Evaluate from technical feasibility, performance, maintainability.
Be brutally honest. Say "this is a bad idea" if it is. No corporate fluff.
STRICT: Reply in exactly 2-3 sentences. Never exceed 3 sentences.`,
	},
	{
		Name:  "Designer",
		Emoji: "🎨",
		System: `You are an opinionated UX designer. 8 years experience. You fight for users.
Evaluate from user experience, accessibility, design consistency.
Push back on developers who ignore UX. Disagree openly when needed.
STRICT: Reply in exactly 2-3 sentences. Never exceed 3 sentences.`,
	},
	{
		Name:  "PM",
		Emoji: "📊",
		System: `You are a no-nonsense product manager. 7 years experience. You care about ROI and deadlines.
Evaluate from business impact, timeline, prioritization.
Cut through idealism with data and deadlines. Be the pragmatic voice.
STRICT: Reply in exactly 2-3 sentences. Never exceed 3 sentences.`,
	},
	{
		Name:  "Security",
		Emoji: "🔒",
		System: `You are a paranoid security engineer. 6 years experience. You assume everything will be exploited.
Evaluate from security, privacy, compliance perspectives.
Block ideas that have unmitigated risks. Be the voice of caution.
STRICT: Reply in exactly 2-3 sentences. Never exceed 3 sentences.`,
	},
}

// Provider interface
type LLMProvider interface {
	Call(system, prompt string, tracker *CostTracker) string
	Name() string
	Model() string
}

// Anthropic provider
type AnthropicProvider struct {
	apiKey string
	model  string
}

func (a *AnthropicProvider) Name() string  { return "Anthropic" }
func (a *AnthropicProvider) Model() string { return a.model }

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *AnthropicProvider) Call(system, prompt string, tracker *CostTracker) string {
	reqBody := claudeRequest{
		Model:     a.model,
		MaxTokens: 150,
		System:    system,
		Messages:  []claudeMessage{{Role: "user", Content: prompt}},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("[Error: %v]", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Sprintf("[API Error %d: %s]", resp.StatusCode, string(body))
	}

	var result claudeResponse
	json.Unmarshal(body, &result)

	tracker.Add(Usage{
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		Provider:     "anthropic",
	})

	if len(result.Content) > 0 {
		return result.Content[0].Text
	}
	return "[No response]"
}

// OpenAI provider
type OpenAIProvider struct {
	apiKey string
	model  string
}

func (o *OpenAIProvider) Name() string  { return "OpenAI" }
func (o *OpenAIProvider) Model() string { return o.model }

type openaiRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_completion_tokens"`
	Messages  []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (o *OpenAIProvider) Call(system, prompt string, tracker *CostTracker) string {
	reqBody := openaiRequest{
		Model:     o.model,
		MaxTokens: 150,
		Messages: []openaiMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("[Error: %v]", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Sprintf("[API Error %d: %s]", resp.StatusCode, string(body))
	}

	var result openaiResponse
	json.Unmarshal(body, &result)

	tracker.Add(Usage{
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		Provider:     "openai",
	})

	if len(result.Choices) > 0 {
		return result.Choices[0].Message.Content
	}
	return "[No response]"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: teamtalk <question>")
		fmt.Println("       teamtalk --demo")
		fmt.Println("")
		fmt.Println("Environment variables:")
		fmt.Println("  ANTHROPIC_API_KEY    Use Claude (default: claude-sonnet-4-20250514)")
		fmt.Println("  OPENAI_API_KEY       Use GPT (default: gpt-4o-mini)")
		fmt.Println("  TEAMTALK_MODEL       Override model name")
		os.Exit(1)
	}

	question := strings.Join(os.Args[1:], " ")

	if question == "--demo" {
		runDemo()
		return
	}

	provider := detectProvider()
	if provider == nil {
		fmt.Println("⚠️  No API key found. Running demo mode.\n")
		fmt.Println("Set ANTHROPIC_API_KEY or OPENAI_API_KEY to use real AI.\n")
		runDemo()
		return
	}

	runDebate(question, provider)
}

func detectProvider() LLMProvider {
	model := os.Getenv("TEAMTALK_MODEL")

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
		return &AnthropicProvider{apiKey: key, model: model}
	}

	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		if model == "" {
			model = "gpt-4o-mini"
		}
		return &OpenAIProvider{apiKey: key, model: model}
	}

	return nil
}

func runDebate(question string, provider LLMProvider) {
	team := defaultTeam
	rounds := 3

	tracker := &CostTracker{
		provider: provider.Name(),
		model:    provider.Model(),
	}

	fmt.Printf("\n🏛️  TeamTalk — AI Team Discussion\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📋 Topic: %s\n", question)
	fmt.Printf("🤖 Provider: %s (%s)\n", provider.Name(), provider.Model())
	fmt.Printf("👥 Team: ")
	for i, p := range team {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%s %s", p.Emoji, p.Name)
	}
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	var allOpinions []Opinion

	for round := 1; round <= rounds; round++ {
		switch round {
		case 1:
			fmt.Printf("📢 Round %d — Initial Opinions\n", round)
		case 2:
			fmt.Printf("\n📢 Round %d — Rebuttals & Challenges\n", round)
		default:
			fmt.Printf("\n📢 Round %d — Final Positions\n", round)
		}
		fmt.Println("─────────────────────────────────")

		opinions := gatherOpinions(team, question, allOpinions, round, provider, tracker)
		allOpinions = append(allOpinions, opinions...)

		for _, o := range opinions {
			printOpinion(o)
		}
	}

	// Final summary
	fmt.Printf("\n⚖️  Final Summary\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	summary := generateSummary(question, allOpinions, provider, tracker)
	typewrite(summary)
	fmt.Println()

	// Print usage
	tracker.Print()
}

func gatherOpinions(team []Persona, question string, history []Opinion, round int, provider LLMProvider, tracker *CostTracker) []Opinion {
	var mu sync.Mutex
	var wg sync.WaitGroup
	opinions := make([]Opinion, len(team))

	for i, persona := range team {
		wg.Add(1)
		go func(idx int, p Persona) {
			defer wg.Done()

			prompt := buildPrompt(p, question, history, round)
			response := provider.Call(p.System, prompt, tracker)

			mu.Lock()
			opinions[idx] = Opinion{
				Persona: p,
				Content: response,
				Round:   round,
			}
			mu.Unlock()
		}(i, persona)
	}

	wg.Wait()
	return opinions
}

func buildPrompt(p Persona, question string, history []Opinion, round int) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Topic for discussion: %s\n\n", question))

	if round == 1 {
		sb.WriteString(fmt.Sprintf("As %s, give your initial take on this topic. Be direct.", p.Name))
	} else if round == 2 {
		sb.WriteString("Previous discussion:\n")
		for _, o := range history {
			sb.WriteString(fmt.Sprintf("[%s %s]: %s\n\n", o.Persona.Emoji, o.Persona.Name, o.Content))
		}
		sb.WriteString(fmt.Sprintf("\nAs %s, challenge at least one other person's point. Disagree where needed. Don't just agree.", p.Name))
	} else {
		sb.WriteString("Previous discussion:\n")
		for _, o := range history {
			sb.WriteString(fmt.Sprintf("[%s %s]: %s\n\n", o.Persona.Emoji, o.Persona.Name, o.Content))
		}
		sb.WriteString(fmt.Sprintf("\nAs %s, give your FINAL position. State clearly: do it, don't do it, or do it with conditions. One sentence.", p.Name))
	}

	return sb.String()
}

func generateSummary(question string, opinions []Opinion, provider LLMProvider, tracker *CostTracker) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic: %s\n\nFull discussion:\n", question))
	for _, o := range opinions {
		sb.WriteString(fmt.Sprintf("[Round %d - %s %s]: %s\n\n", o.Round, o.Persona.Emoji, o.Persona.Name, o.Content))
	}
	sb.WriteString("\nSummarize the key agreements, disagreements, and final recommendation in 3-5 bullet points.")

	system := "You are a skilled meeting facilitator. Summarize team discussions into clear, actionable conclusions."
	return provider.Call(system, sb.String(), tracker)
}

func printOpinion(o Opinion) {
	fmt.Printf("\n%s %s:\n", o.Persona.Emoji, o.Persona.Name)
	typewrite(o.Content)
	fmt.Println()
}

func typewrite(text string) {
	for _, ch := range text {
		fmt.Print(string(ch))
		time.Sleep(8 * time.Millisecond)
	}
}

func runDemo() {
	question := "Should we add a WebSocket real-time feature to our REST API?"

	fmt.Printf("\n🏛️  TeamTalk — AI Team Discussion\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📋 Topic: %s\n", question)
	fmt.Printf("👥 Team: 💻 Developer, 🎨 Designer, 📊 PM, 🔒 Security\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Println("📢 Round 1 — Initial Opinions")
	fmt.Println("─────────────────────────────────")

	demoOpinions := []struct {
		emoji, name, text string
	}{
		{"💻", "Developer", "WebSocket adds significant complexity — we need connection state management, heartbeats, reconnection logic, and horizontal scaling becomes harder since sticky sessions or a pub/sub layer (Redis) is required. If we only need real-time for notifications, SSE is simpler and works over HTTP/2. I'd push back unless we have a strong bidirectional use case."},
		{"🎨", "Designer", "Real-time updates dramatically improve perceived performance. Users hate clicking refresh. For dashboards and collaborative features, WebSocket enables live cursors, typing indicators, and instant syncs that users now expect. But we need clear loading and disconnection states — don't just silently fail."},
		{"📊", "PM", "Three of our top 5 feature requests involve real-time data. Competitors shipped live dashboards last quarter. The question isn't if, but when. I'd scope a Phase 1 with just notifications/status updates via SSE, then evaluate WebSocket for Phase 2 collaborative features based on adoption data."},
		{"🔒", "Security", "WebSocket connections bypass traditional HTTP middleware — auth tokens need to be validated on upgrade AND periodically during the connection. Rate limiting is harder. We need message-level validation, not just connection-level. Also, wss:// only — no plain ws:// in production."},
	}

	for _, o := range demoOpinions {
		fmt.Printf("\n%s %s:\n", o.emoji, o.name)
		typewrite(o.text)
		fmt.Println()
	}

	fmt.Printf("\n📢 Round 2 — Rebuttals & Responses\n")
	fmt.Println("─────────────────────────────────")

	demoRebuttals := []struct {
		emoji, name, text string
	}{
		{"💻", "Developer", "PM's phased approach makes sense. SSE for Phase 1 cuts our complexity in half — no connection state, no sticky sessions. If we use a clean event interface now, migrating to WebSocket later is straightforward. I can prototype SSE in 3 days."},
		{"🎨", "Designer", "Agreed on phasing. For Phase 1, I need at minimum: live notification badges, optimistic UI updates, and a visible connection status indicator. These alone will cover 80% of the UX improvement users are asking for."},
		{"📊", "PM", "Love the 3-day prototype timeline. Let's ship SSE notifications in Sprint 14, measure engagement lift, then decide Phase 2 scope. Developer, can you draft the event interface spec by Friday?"},
		{"🔒", "Security", "SSE is better for Phase 1 security-wise — it's just HTTP with standard auth. For the eventual WebSocket phase, I'll prepare an auth spec with token refresh over the connection. Let's also add connection audit logging from day one."},
	}

	for _, o := range demoRebuttals {
		fmt.Printf("\n%s %s:\n", o.emoji, o.name)
		typewrite(o.text)
		fmt.Println()
	}

	fmt.Printf("\n⚖️  Final Summary\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	summary := `• ✅ Consensus: Phase 1 with SSE (Server-Sent Events) for notifications — simpler, secure, 3-day prototype
• ✅ Scope: Live notification badges, optimistic UI updates, connection status indicator
• ⏳ Phase 2: Evaluate WebSocket for collaborative features based on Phase 1 adoption metrics
• 🔒 Security: Standard HTTP auth for SSE; prepare WebSocket auth spec in parallel
• 📅 Timeline: Event interface spec by Friday → SSE shipped in Sprint 14`

	typewrite(summary)
	fmt.Println()

	fmt.Printf("\n📊 Token Usage\n")
	fmt.Println("─────────────────────────────────")
	fmt.Println("   Provider:  Demo (no API calls)")
	fmt.Println("   Cost:      $0.0000")
}
