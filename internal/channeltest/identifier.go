package channeltest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// KeywordEntry maps a pattern to a channel identification.
type KeywordEntry struct {
	Pattern string   `json:"pattern"`
	Channel string   `json:"channel"`
	Scopes  []string `json:"scopes"`
}

// CustomKeyword is a user-defined keyword entry persisted in DB.
type CustomKeyword struct {
	ID        string    `json:"id"`
	Pattern   string    `json:"pattern"`
	Channel   string    `json:"channel"`
	Scopes    []string  `json:"scopes"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// ChannelHit records a keyword match during channel identification.
type ChannelHit struct {
	Keyword string `json:"keyword"`
	Channel string `json:"channel"`
	Source  string `json:"source"`
	Context string `json:"context"`
}

var defaultKeywords = []KeywordEntry{
	{Pattern: "kiro", Channel: "Amazon Kiro", Scopes: []string{"body", "thinking", "headers"}},
	{Pattern: "antigravity", Channel: "Amazon Antigravity", Scopes: []string{"body", "thinking"}},
	{Pattern: "msg_bdrk_", Channel: "AWS Bedrock", Scopes: []string{"body"}},
	{Pattern: "x-amzn-requestid", Channel: "AWS (header)", Scopes: []string{"headers"}},
	{Pattern: "x-amzn-bedrock", Channel: "AWS Bedrock (header)", Scopes: []string{"headers"}},
	{Pattern: "gen-", Channel: "OpenRouter", Scopes: []string{"id_prefix"}},
	{Pattern: "chatcmpl-", Channel: "OneAPI/sub2api", Scopes: []string{"id_prefix"}},
	{Pattern: "x-new-api-version", Channel: "NewAPI/sub2api", Scopes: []string{"headers"}},
	{Pattern: "openrouter", Channel: "OpenRouter", Scopes: []string{"body", "headers"}},
	{Pattern: "one-api", Channel: "OneAPI", Scopes: []string{"headers"}},
	{Pattern: "new-api", Channel: "NewAPI", Scopes: []string{"headers"}},
	{Pattern: "x-oneapi", Channel: "OneAPI (header)", Scopes: []string{"headers"}},
}

// KeywordPersist holds callbacks for SQLite persistence of keywords.
type KeywordPersist struct {
	DB      *sql.DB
	LogErr  func(op string, err error)
	Save    func(db *sql.DB, kw *CustomKeyword) error
	Delete  func(db *sql.DB, id string) error
	LoadAll func(db *sql.DB) ([]*CustomKeyword, error)
}

// KeywordStore manages the merged set of built-in + custom keywords.
type KeywordStore struct {
	mu      sync.RWMutex
	persist *KeywordPersist
	custom  map[string]*CustomKeyword
}

// NewKeywordStore creates a keyword store, loading existing custom keywords from DB.
func NewKeywordStore(p *KeywordPersist) *KeywordStore {
	s := &KeywordStore{
		persist: p,
		custom:  make(map[string]*CustomKeyword),
	}
	if p != nil && p.DB != nil && p.LoadAll != nil {
		if keywords, err := p.LoadAll(p.DB); err == nil {
			for _, kw := range keywords {
				s.custom[kw.ID] = kw
			}
		}
	}
	return s
}

// AllKeywords returns the merged list: built-in defaults + enabled custom keywords.
func (s *KeywordStore) AllKeywords() []KeywordEntry {
	out := make([]KeywordEntry, len(defaultKeywords))
	copy(out, defaultKeywords)

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, kw := range s.custom {
		if kw.Enabled {
			out = append(out, KeywordEntry{
				Pattern: kw.Pattern,
				Channel: kw.Channel,
				Scopes:  kw.Scopes,
			})
		}
	}
	return out
}

// ListCustom returns all custom keywords (enabled and disabled).
func (s *KeywordStore) ListCustom() []*CustomKeyword {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*CustomKeyword, 0, len(s.custom))
	for _, kw := range s.custom {
		out = append(out, kw)
	}
	return out
}

// ListAll returns built-in + custom keywords for display.
func (s *KeywordStore) ListAll() map[string]any {
	return map[string]any{
		"builtin": defaultKeywords,
		"custom":  s.ListCustom(),
	}
}

// Add creates or updates a custom keyword.
func (s *KeywordStore) Add(kw *CustomKeyword) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if kw.ID == "" {
		kw.ID = newKWID()
	}
	if kw.CreatedAt.IsZero() {
		kw.CreatedAt = time.Now()
	}
	if p := s.persist; p != nil && p.DB != nil && p.Save != nil {
		if err := p.Save(p.DB, kw); err != nil && p.LogErr != nil {
			p.LogErr("save_keyword", err)
		}
	}
	s.custom[kw.ID] = kw
}

// Delete removes a custom keyword by ID.
func (s *KeywordStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.custom[id]; !ok {
		return false
	}
	if p := s.persist; p != nil && p.DB != nil && p.Delete != nil {
		if err := p.Delete(p.DB, id); err != nil && p.LogErr != nil {
			p.LogErr("delete_keyword", err)
		}
	}
	delete(s.custom, id)
	return true
}

func newKWID() string {
	return "kw_" + time.Now().Format("20060102150405") + "_" + randomHex(4)
}

// exchangeScan holds per-exchange scan targets for keyword matching.
type exchangeScan struct {
	requestLower  string
	bodyTexts     []string
	thinkingTexts []string
	headerTexts   []string
	idPrefixes    []string
}

func extractPerExchange(exchanges []Exchange) []exchangeScan {
	var out []exchangeScan
	for _, ex := range exchanges {
		scan := exchangeScan{
			requestLower: strings.ToLower(string(ex.Request)),
		}
		for key, vals := range ex.ResponseHeaders {
			for _, v := range vals {
				scan.headerTexts = append(scan.headerTexts, strings.ToLower(key)+": "+strings.ToLower(v))
			}
		}
		if len(ex.Response) == 0 {
			out = append(out, scan)
			continue
		}
		raw := string(ex.Response)
		scan.bodyTexts = append(scan.bodyTexts, raw)

		var resp map[string]any
		if json.Unmarshal(ex.Response, &resp) == nil {
			if id, _ := resp["id"].(string); id != "" {
				scan.idPrefixes = append(scan.idPrefixes, id)
			}
			if content, ok := resp["content"].([]any); ok {
				for _, cb := range content {
					m, ok := cb.(map[string]any)
					if !ok {
						continue
					}
					if t, _ := m["type"].(string); t == "thinking" {
						if text, _ := m["thinking"].(string); text != "" {
							scan.thinkingTexts = append(scan.thinkingTexts, text)
						}
					}
				}
			}
		}
		out = append(out, scan)
	}
	return out
}

// channelFP defines a check-result fingerprint for auto-detecting a channel type.
type channelFP struct {
	Channel  string
	Profile  string
	MustPass []string
	MustFail []string
}

var channelFingerprints = []channelFP{
	{
		Channel:  "Anthropic Console 直连",
		Profile:  "console",
		MustPass: []string{"headers", "cf_headers", "server_timing", "id_format"},
	},
	{
		Channel:  "Max 订阅",
		Profile:  "max",
		MustPass: []string{"id_format", "sse_done", "container", "bedrock_state"},
		MustFail: []string{"headers", "cf_headers", "server_timing"},
	},
	{
		Channel:  "AWS Bedrock",
		Profile:  "bedrock",
		MustPass: []string{"sse_done"},
		MustFail: []string{"id_format"},
	},
	{
		Channel:  "OpenAI 兼容层代理",
		MustFail: []string{"sse_done", "id_format"},
	},
}

func matchFingerprints(checks []CheckResult) []ChannelHit {
	checkPass := make(map[string]bool)
	checkSeen := make(map[string]bool)
	for _, c := range checks {
		if !checkSeen[c.Name] {
			checkPass[c.Name] = c.Pass
			checkSeen[c.Name] = true
		} else if checkPass[c.Name] && !c.Pass {
			checkPass[c.Name] = false
		}
	}

	var hits []ChannelHit
	for _, fp := range channelFingerprints {
		ok := true
		matched := 0
		for _, name := range fp.MustPass {
			if !checkSeen[name] {
				ok = false
				break
			}
			matched++
			if !checkPass[name] {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		for _, name := range fp.MustFail {
			if !checkSeen[name] {
				ok = false
				break
			}
			matched++
			if checkPass[name] {
				ok = false
				break
			}
		}
		if !ok || matched == 0 {
			continue
		}
		hits = append(hits, ChannelHit{
			Keyword: "fingerprint:" + fp.Profile,
			Channel: fp.Channel,
			Source:  "check_fingerprint",
			Context: fmt.Sprintf("%d/%d rules matched → profile: %s", matched, matched, fp.Profile),
		})
	}
	return hits
}

// IdentifyChannel scans exchanges for keyword matches and check fingerprints.
// Uses the provided keyword store if non-nil, otherwise falls back to defaults.
func IdentifyChannel(exchanges []Exchange, checks []CheckResult, kwStore ...*KeywordStore) []ChannelHit {
	var hits []ChannelHit
	seen := map[string]bool{}

	addHit := func(keyword, channel, source, ctx string) {
		key := keyword + "|" + source
		if seen[key] {
			return
		}
		seen[key] = true
		hits = append(hits, ChannelHit{
			Keyword: keyword,
			Channel: channel,
			Source:  source,
			Context: truncate(ctx, 100),
		})
	}

	keywords := defaultKeywords
	if len(kwStore) > 0 && kwStore[0] != nil {
		keywords = kwStore[0].AllKeywords()
	}

	exScans := extractPerExchange(exchanges)

	for _, kw := range keywords {
		kwLower := strings.ToLower(kw.Pattern)
		for _, scope := range kw.Scopes {
			switch scope {
			case "body":
				for _, es := range exScans {
					if strings.Contains(es.requestLower, kwLower) {
						continue
					}
					for _, text := range es.bodyTexts {
						if idx := strings.Index(strings.ToLower(text), kwLower); idx >= 0 {
							start := idx - 30
							if start < 0 {
								start = 0
							}
							end := idx + len(kw.Pattern) + 30
							if end > len(text) {
								end = len(text)
							}
							addHit(kw.Pattern, kw.Channel, "response_body", text[start:end])
						}
					}
				}
			case "thinking":
				for _, es := range exScans {
					if strings.Contains(es.requestLower, kwLower) {
						continue
					}
					for _, text := range es.thinkingTexts {
						if idx := strings.Index(strings.ToLower(text), kwLower); idx >= 0 {
							start := idx - 30
							if start < 0 {
								start = 0
							}
							end := idx + len(kw.Pattern) + 30
							if end > len(text) {
								end = len(text)
							}
							addHit(kw.Pattern, kw.Channel, "thinking_block", text[start:end])
						}
					}
				}
			case "headers":
				for _, es := range exScans {
					for _, text := range es.headerTexts {
						if strings.Contains(text, kwLower) {
							addHit(kw.Pattern, kw.Channel, "response_headers", text)
						}
					}
				}
			case "id_prefix":
				for _, es := range exScans {
					for _, id := range es.idPrefixes {
						if strings.HasPrefix(id, kw.Pattern) {
							addHit(kw.Pattern, kw.Channel, "message_id", id)
						}
					}
				}
			}
		}
	}

	for _, c := range checks {
		if c.Name == "backend_type" && !c.Pass && c.Detail != "" {
			detail := strings.ToLower(c.Detail)
			if strings.Contains(detail, "bedrock") {
				addHit("backend_type:bedrock", "AWS Bedrock", "check_result", c.Detail)
			} else if strings.Contains(detail, "openrouter") {
				addHit("backend_type:openrouter", "OpenRouter", "check_result", c.Detail)
			} else if strings.Contains(detail, "oneapi") || strings.Contains(detail, "sub2api") {
				addHit("backend_type:oneapi", "OneAPI/sub2api", "check_result", c.Detail)
			}
		}
	}

	hits = append(hits, matchFingerprints(checks)...)
	return hits
}
