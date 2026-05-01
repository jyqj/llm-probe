package fingerprint

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"
)

// RateLimitState holds simulated rate limit info for response headers.
type RateLimitState struct {
	OrgID        string
	InputLimit   int
	OutputLimit  int
	TokensLimit  int
	ReqLimit     int
	InputRemain  int
	OutputRemain int
	TokensRemain int
	ReqRemain    int
}

type orgEntry struct {
	orgID        string
	inputUsed    int
	outputUsed   int
	reqCount     int
	window       []windowEntry
	lastReset    time.Time
	baseinPct    float64
	baseoutPct   float64
}

type windowEntry struct {
	t   time.Time
	in  int
	out int
}

var (
	orgPool     []*orgEntry
	orgPoolOnce sync.Once
	rlMu        sync.Mutex
)

func initOrgPool() {
	orgPoolOnce.Do(func() {
		for i := 0; i < 7; i++ {
			orgPool = append(orgPool, &orgEntry{
				orgID:      newUUID(),
				baseinPct:  randFloat(0.05, 0.35),
				baseoutPct: randFloat(0.02, 0.10),
				lastReset:  time.Now().Truncate(time.Minute),
			})
		}
	})
}

func randFloat(lo, hi float64) float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(10000))
	return lo + (hi-lo)*float64(n.Int64())/10000.0
}

func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(v.Int64())
}

// RateLimitTick simulates rate limit state for a request.
// Uses a stateful sliding window per org to produce realistic consumption curves.
func RateLimitTick(model string, consumedInput, consumedOutput int) *RateLimitState {
	initOrgPool()
	rlMu.Lock()
	defer rlMu.Unlock()

	now := time.Now()
	isHaiku := containsCI(model, "haiku")

	limits := struct{ inLim, outLim, tokLim, reqLim int }{
		2_000_000, 400_000, 2_400_000, 4000,
	}
	if isHaiku {
		limits.inLim = 4_000_000
		limits.outLim = 800_000
		limits.tokLim = 4_800_000
	}

	org := orgPool[randIntn(len(orgPool))]

	// Reset window on minute boundary (mimics official behavior)
	currentMinute := now.Truncate(time.Minute)
	if currentMinute.After(org.lastReset) {
		// Decay: retain ~30% of accumulated usage across reset
		org.inputUsed = int(float64(org.inputUsed) * 0.3)
		org.outputUsed = int(float64(org.outputUsed) * 0.3)
		org.reqCount = int(float64(org.reqCount) * 0.3)
		org.lastReset = currentMinute
	}

	// Slide window: remove entries older than 60s
	cutoff := now.Add(-60 * time.Second)
	kept := org.window[:0]
	for _, w := range org.window {
		if w.t.After(cutoff) {
			kept = append(kept, w)
		}
	}
	org.window = append(kept, windowEntry{now, consumedInput, consumedOutput})

	// Accumulate actual consumption
	org.inputUsed += consumedInput
	org.outputUsed += consumedOutput
	org.reqCount++

	// Calculate remaining with base load + small drift
	baseIn := int(float64(limits.inLim) * org.baseinPct)
	baseOut := int(float64(limits.outLim) * org.baseoutPct)
	driftIn := int(float64(baseIn) * randFloat(-0.05, 0.05))
	driftOut := int(float64(baseOut) * randFloat(-0.05, 0.05))

	effIn := org.inputUsed + baseIn + driftIn
	effOut := org.outputUsed + baseOut + driftOut
	effReq := org.reqCount + randIntn(5)

	return &RateLimitState{
		OrgID:        org.orgID,
		InputLimit:   limits.inLim,
		OutputLimit:  limits.outLim,
		TokensLimit:  limits.tokLim,
		ReqLimit:     limits.reqLim,
		InputRemain:  max(0, limits.inLim-effIn),
		OutputRemain: max(0, limits.outLim-effOut),
		TokensRemain: max(0, limits.tokLim-effIn-effOut),
		ReqRemain:    max(0, limits.reqLim-effReq),
	}
}

func containsCI(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			c := s[i+j]
			if c >= 'A' && c <= 'Z' {
				c += 32
			}
			d := sub[j]
			if d >= 'A' && d <= 'Z' {
				d += 32
			}
			if c != d {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
