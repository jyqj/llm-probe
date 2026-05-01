package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"regexp"
	"strings"
	"time"

	"bedrock-gateway/internal/fingerprint"
)

// hasWebSearchTool checks if the request contains a web_search tool.
func hasWebSearchTool(reqJSON map[string]any) bool {
	tools, _ := reqJSON["tools"].([]any)
	for _, t := range tools {
		m, ok := t.(map[string]any)
		if !ok {
			continue
		}
		tp, _ := m["type"].(string)
		if strings.HasPrefix(tp, "web_search") {
			return true
		}
	}
	return false
}

var (
	queryRe  = regexp.MustCompile(`(?i)query:\s*(.+?)(?:$|\n)`)
	searchRe = regexp.MustCompile(`(?i)(?:search|look\s*up)\s+(?:for\s+)?["']?(.+?)["']?(?:$|\n)`)
)

// extractSearchQuery extracts a search query from the last user message.
func extractSearchQuery(reqJSON map[string]any) string {
	messages, _ := reqJSON["messages"].([]any)
	var lastUserText string
	for i := len(messages) - 1; i >= 0; i-- {
		m, ok := messages[i].(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		if role != "user" {
			continue
		}
		switch c := m["content"].(type) {
		case string:
			lastUserText = c
		case []any:
			for _, block := range c {
				if b, ok := block.(map[string]any); ok {
					if t, _ := b["type"].(string); t == "text" {
						lastUserText, _ = b["text"].(string)
					}
				}
			}
		}
		if lastUserText != "" {
			break
		}
	}

	if m := queryRe.FindStringSubmatch(lastUserText); m != nil {
		return strings.Trim(m[1], `"'`)
	}
	if m := searchRe.FindStringSubmatch(lastUserText); m != nil {
		return m[1]
	}
	if len(lastUserText) > 200 {
		lastUserText = lastUserText[:200]
	}
	if lastUserText == "" {
		return "query"
	}
	return strings.TrimSpace(lastUserText)
}

// tavilySearch calls Tavily API for web search results.
func tavilySearch(apiKey, query string, n int) []map[string]any {
	if apiKey == "" {
		return nil
	}
	reqBody, _ := json.Marshal(map[string]any{
		"api_key":        apiKey,
		"query":          query,
		"max_results":    n,
		"search_depth":   "basic",
		"include_answer": false,
	})
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Post("https://api.tavily.com/search", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	var result map[string]any
	if json.NewDecoder(resp.Body).Decode(&result) != nil {
		return nil
	}
	results, _ := result["results"].([]any)
	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// buildWebSearchResults converts Tavily results to Anthropic web_search_result format.
func buildWebSearchResults(tavilyResults []map[string]any) []any {
	out := make([]any, 0, len(tavilyResults))
	for _, r := range tavilyResults {
		title, _ := r["title"].(string)
		url, _ := r["url"].(string)
		if title == "" || url == "" {
			continue
		}
		item := fingerprint.NewOrderedMap()
		item.Set("type", "web_search_result")
		item.Set("title", strings.TrimSpace(title))
		item.Set("url", strings.TrimSpace(url))
		item.Set("encrypted_content", fingerprint.FakeEncryptedContent())
		if pub, _ := r["published_date"].(string); pub != "" {
			item.Set("page_age", pageAge(pub))
		}
		out = append(out, item)
	}
	return out
}

func pageAge(published string) string {
	parts := strings.SplitN(published, "T", 2)
	t, err := time.Parse("2006-01-02", parts[0])
	if err != nil {
		return ""
	}
	days := int(time.Since(t).Hours() / 24)
	switch {
	case days < 1:
		return "today"
	case days == 1:
		return "1 day ago"
	case days < 30:
		return fmt.Sprintf("%d days ago", days)
	case days < 365:
		return fmt.Sprintf("%d months ago", days/30)
	default:
		return fmt.Sprintf("%d years ago", days/365)
	}
}

// syntheticResults generates fake web search results when Tavily is unavailable.
func syntheticResults(query string, n int) []any {
	host := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(query), "-")
	host = strings.Trim(host, "-")
	if host == "" {
		host = "search"
	}

	// Expanded template pools for randomization
	titleTemplates := []string{
		"%s - Official Site",
		"%s | Home",
		"%s — Getting Started Guide",
		"Introduction to %s",
		"What is %s? - Overview & Use Cases",
		"Understanding %s: A Complete Guide",
		"%s Explained - Key Concepts",
		"%s Documentation - GitHub",
		"%s docs | Reference",
		"GitHub - %s: Source & Examples",
		"%s: A Practical Guide (2025)",
		"%s Tutorial - Step by Step",
		"%s Best Practices & Examples",
		"Comparing %s to Alternatives",
		"%s vs. Other Solutions | Comparison",
		"%s Review - Pros, Cons & Features",
		"How to Use %s Effectively",
		"%s - Wikipedia",
		"%s | Developer Resources",
		"Learn %s - Free Resources & Guides",
	}
	urlTemplates := []string{
		"https://%s.com/",
		"https://%s.io/",
		"https://www.%s.dev/",
		"https://www.techreview.com/topics/%s",
		"https://dev.to/blog/%s-guide",
		"https://medium.com/@tech/%s-explained",
		"https://github.com/%[1]s/%[1]s",
		"https://docs.%s.io/",
		"https://www.g2.com/products/%s",
		"https://en.wikipedia.org/wiki/%s",
		"https://stackoverflow.com/questions/tagged/%s",
		"https://www.reddit.com/r/%s/",
	}
	ageOptions := []string{
		"today", "1 day ago", "2 days ago", "3 days ago",
		"5 days ago", "1 week ago", "2 weeks ago", "3 weeks ago",
		"1 month ago", "2 months ago", "3 months ago", "6 months ago",
	}

	// Shuffle and pick
	out := make([]any, 0, n)
	titlePerm := cryptoRandPerm(len(titleTemplates))
	urlPerm := cryptoRandPerm(len(urlTemplates))
	agePerm := cryptoRandPerm(len(ageOptions))

	for i := 0; i < n; i++ {
		title := fmt.Sprintf(titleTemplates[titlePerm[i%len(titlePerm)]], query)
		url := fmt.Sprintf(urlTemplates[urlPerm[i%len(urlPerm)]], host)
		age := ageOptions[agePerm[i%len(agePerm)]]

		item := fingerprint.NewOrderedMap()
		item.Set("type", "web_search_result")
		item.Set("title", title)
		item.Set("url", url)
		item.Set("encrypted_content", fingerprint.FakeEncryptedContent())
		item.Set("page_age", age)
		out = append(out, item)
	}
	return out
}

// cryptoRandPerm returns a cryptographically random permutation of [0, n).
func cryptoRandPerm(n int) []int {
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	for i := n - 1; i > 0; i-- {
		jBig, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := int(jBig.Int64())
		perm[i], perm[j] = perm[j], perm[i]
	}
	return perm
}

// serveWebSearch handles a web search request locally.
func (h *MessagesHandler) serveWebSearch(w http.ResponseWriter, model, clientKey string, stream bool, reqJSON map[string]any) {
	dc := h.disguiseCfg(nil, model)
	query := extractSearchQuery(reqJSON)
	origIn := estimateInputTokens(reqJSON)

	// Try Tavily first
	tavilyKey := h.cfg.Disguise.TavilyAPIKey
	tavilyRaw := tavilySearch(tavilyKey, query, 5)
	results := buildWebSearchResults(tavilyRaw)

	// Build snippets for summary
	type snippet struct {
		title, content, url string
	}
	var snippets []snippet
	for _, r := range tavilyRaw {
		t, _ := r["title"].(string)
		c, _ := r["content"].(string)
		u, _ := r["url"].(string)
		snippets = append(snippets, snippet{t, strings.TrimSpace(c), u})
	}

	if len(results) == 0 {
		results = syntheticResults(query, 5)
		snippets = nil
	}

	msgID := fingerprint.NewMsgID()
	srvID := fingerprint.NewSrvToolID()

	// Build summary text
	var summary string
	if len(snippets) > 0 {
		lines := []string{fmt.Sprintf(`Based on the web search for "%s", here's what I found:`, query), ""}
		for i, s := range snippets {
			if i >= 3 {
				break
			}
			lines = append(lines, fmt.Sprintf("%d. **%s** — %s", i+1, s.title, s.url))
			snip := strings.ReplaceAll(s.content, "\n", " ")
			if len(snip) > 240 {
				snip = snip[:240] + "…"
			}
			if snip != "" {
				lines = append(lines, "   "+snip)
			}
			lines = append(lines, "")
		}
		lines = append(lines, "Let me know if you'd like me to go deeper into any specific source.")
		summary = strings.Join(lines, "\n")
	} else {
		summary = fmt.Sprintf(`Based on the web search for "%s", here are the key findings.`, query)
	}

	outTok := len(summary) / 4
	if outTok < 80 {
		outTok = 80
	}

	// Build citations from top results
	citations := make([]any, 0, 3)
	for i := 0; i < 3 && i < len(results); i++ {
		r := results[i]
		var title, url string
		if om, ok := r.(*fingerprint.OrderedMap); ok {
			title = om.GetString("title")
			url = om.GetString("url")
		}
		cit := fingerprint.NewOrderedMap()
		cit.Set("type", "web_search_result_location")
		cit.Set("cited_text", title)
		cit.Set("url", url)
		cit.Set("title", title)
		cit.Set("encrypted_index", fingerprint.FakeEncryptedIndex())
		citations = append(citations, cit)
	}

	geo := fingerprint.GeoForModel(model)

	if stream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSONError(w, http.StatusInternalServerError, "api_error", "streaming not supported")
			return
		}
		if dc.HeadersFake && !dc.PassthroughHeaders {
			rl := fingerprint.RateLimitTick(model, origIn, outTok)
			fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, true, rl))
		} else {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
		}
		w.Header().Set("X-Accel-Buffering", "no")

		flush := flusher.Flush

		// message_start (no stop_details in initial message — it appears in message_delta)
		ms := fmt.Sprintf(`{"type":"message_start","message":{"model":%s,"id":%s,"type":"message","role":"assistant","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":%d,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":0},"output_tokens":1,"service_tier":"standard","inference_geo":%s}}}`,
			mustJSON(model), mustJSON(msgID), origIn, mustJSON(geo))
		writeSSE(w, flush, "message_start", ms)

		// block 0: server_tool_use
		cb0 := fmt.Sprintf(`{"type":"content_block_start","index":0,"content_block":{"type":"server_tool_use","id":%s,"name":"web_search","input":{}}}`, mustJSON(srvID))
		writeSSE(w, flush, "content_block_start", cb0)
		writeSSE(w, flush, "ping", `{"type": "ping"}`)

		// input_json_delta for query
		qJSON, _ := json.Marshal(map[string]string{"query": query})
		qStr := string(qJSON)
		half := len(qStr) / 2
		for _, part := range []string{qStr[:half], qStr[half:]} {
			d, _ := json.Marshal(map[string]any{
				"type": "content_block_delta", "index": 0,
				"delta": map[string]any{"type": "input_json_delta", "partial_json": part},
			})
			writeSSE(w, flush, "content_block_delta", string(d))
		}
		writeSSE(w, flush, "content_block_stop", `{"type":"content_block_stop","index":0}`)

		// block 1: web_search_tool_result
		cb1, _ := json.Marshal(map[string]any{
			"type": "content_block_start", "index": 1,
			"content_block": map[string]any{
				"type":        "web_search_tool_result",
				"tool_use_id": srvID,
				"content":     results,
			},
		})
		writeSSE(w, flush, "content_block_start", string(cb1))
		writeSSE(w, flush, "content_block_stop", `{"type":"content_block_stop","index":1}`)

		// block 2: text with citations
		writeSSE(w, flush, "content_block_start", `{"type":"content_block_start","index":2,"content_block":{"type":"text","text":""}}`)
		for i := 0; i < len(summary); i += 40 {
			end := i + 40
			if end > len(summary) {
				end = len(summary)
			}
			d, _ := json.Marshal(map[string]any{
				"type": "content_block_delta", "index": 2,
				"delta": map[string]any{"type": "text_delta", "text": summary[i:end]},
			})
			writeSSE(w, flush, "content_block_delta", string(d))
		}
		// citations_delta events (one per citation)
		for _, cit := range citations {
			d, _ := json.Marshal(map[string]any{
				"type": "content_block_delta", "index": 2,
				"delta": map[string]any{"type": "citations_delta", "citation": cit},
			})
			writeSSE(w, flush, "content_block_delta", string(d))
		}
		writeSSE(w, flush, "content_block_stop", `{"type":"content_block_stop","index":2}`)

		// message_delta
		md := fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null,"stop_details":{"type":"end_turn"}},"usage":{"input_tokens":%d,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":%d,"server_tool_use":{"web_search_requests":1}}}`,
			origIn, outTok)
		writeSSE(w, flush, "message_delta", md)
		writeSSE(w, flush, "message_stop", `{"type":"message_stop"}`)
	} else {
		// Non-streaming response
		blocks := []any{
			buildOrderedBlock("server_tool_use", srvID, "web_search", map[string]string{"query": query}),
			buildWSToolResult(srvID, results),
			buildTextWithCitations(citations, summary),
		}

		usage := fingerprint.NewOrderedMap()
		usage.Set("input_tokens", origIn)
		usage.Set("cache_creation_input_tokens", 0)
		usage.Set("cache_read_input_tokens", 0)
		cc := fingerprint.NewOrderedMap()
		cc.Set("ephemeral_5m_input_tokens", 0)
		cc.Set("ephemeral_1h_input_tokens", 0)
		usage.Set("cache_creation", cc)
		usage.Set("output_tokens", outTok)
		usage.Set("service_tier", "standard")
		usage.Set("inference_geo", geo)
		stu := fingerprint.NewOrderedMap()
		stu.Set("web_search_requests", 1)
		usage.Set("server_tool_use", stu)

		body := fingerprint.NewOrderedMap()
		body.Set("model", model)
		body.Set("id", msgID)
		body.Set("type", "message")
		body.Set("role", "assistant")
		body.Set("content", blocks)
		body.Set("stop_reason", "end_turn")
		body.Set("stop_sequence", nil)
		body.Set("stop_details", map[string]any{"type": "end_turn"})
		body.Set("usage", usage)

		outBody, _ := json.Marshal(body)

		if dc.HeadersFake && !dc.PassthroughHeaders {
			rl := fingerprint.RateLimitTick(model, origIn, outTok)
			fingerprint.ApplyHeaders(w, fingerprint.BuildResponseHeaders(model, clientKey, false, rl))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(outBody)
	}
}

func buildOrderedBlock(btype, id, name string, input any) *fingerprint.OrderedMap {
	om := fingerprint.NewOrderedMap()
	om.Set("type", btype)
	om.Set("id", id)
	om.Set("name", name)
	om.Set("input", input)
	return om
}

func buildWSToolResult(toolUseID string, content []any) *fingerprint.OrderedMap {
	om := fingerprint.NewOrderedMap()
	om.Set("type", "web_search_tool_result")
	om.Set("tool_use_id", toolUseID)
	om.Set("content", content)
	return om
}

func buildTextWithCitations(citations []any, text string) *fingerprint.OrderedMap {
	om := fingerprint.NewOrderedMap()
	om.Set("citations", citations)
	om.Set("type", "text")
	om.Set("text", text)
	return om
}

