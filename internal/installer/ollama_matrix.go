package installer

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ModelRec is a single recommendation entry.
type ModelRec struct {
	Name  string   // ollama model tag, e.g. "gemma2:2b"
	Tasks []string // chat, embed, classify
}

// TierKey identifies a RAM tier.
type TierKey string

const (
	TierNone     TierKey = "none"     // <4 GB
	TierTiny     TierKey = "tiny"     // 4-6 GB
	TierSmall    TierKey = "small"    // 6-8 GB
	TierBalanced TierKey = "balanced" // 8-12 GB
	TierMedium   TierKey = "medium"   // 12-16 GB
	TierLarge    TierKey = "large"    // 16-24 GB
	TierXL       TierKey = "xl"       // 24-32 GB
	TierXXL      TierKey = "xxl"      // >32 GB
)

// RecommendationMatrix is keyed by nSelf CLI version. Only the current version
// entry is authoritative at runtime; older entries are kept for historical
// reproducibility.
var RecommendationMatrix = map[string]map[TierKey][]ModelRec{
	"1.0.3": {
		TierNone: {},
		TierTiny: {
			{Name: "gemma2:2b", Tasks: []string{"chat", "classify"}},
			{Name: "nomic-embed-text", Tasks: []string{"embed"}},
		},
		TierSmall: {
			{Name: "phi3:mini", Tasks: []string{"chat", "classify"}},
			{Name: "nomic-embed-text", Tasks: []string{"embed"}},
		},
		TierBalanced: {
			{Name: "qwen2.5:3b", Tasks: []string{"chat"}},
			{Name: "nomic-embed-text", Tasks: []string{"embed"}},
			{Name: "gemma2:2b", Tasks: []string{"classify"}},
		},
		TierMedium: {
			{Name: "llama3.2:3b", Tasks: []string{"chat"}},
			{Name: "nomic-embed-text", Tasks: []string{"embed"}},
			{Name: "gemma2:2b", Tasks: []string{"classify"}},
		},
		TierLarge: {
			{Name: "qwen2.5:7b", Tasks: []string{"chat"}},
			{Name: "nomic-embed-text", Tasks: []string{"embed"}},
			{Name: "gemma2:2b", Tasks: []string{"classify"}},
		},
		TierXL: {
			{Name: "llama3.1:8b", Tasks: []string{"chat"}},
			{Name: "bge-large", Tasks: []string{"embed"}},
			{Name: "gemma2:2b", Tasks: []string{"classify"}},
		},
		TierXXL: {
			{Name: "qwen2.5:14b", Tasks: []string{"chat"}},
			{Name: "bge-large", Tasks: []string{"embed"}},
			{Name: "gemma2:2b", Tasks: []string{"classify"}},
		},
	},
}

// MemAvailableMB reads /proc/meminfo on Linux and returns the MemAvailable
// value in MB. Returns 0 and an error on non-Linux or read failure.
func MemAvailableMB() (int, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("read meminfo: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return 0, fmt.Errorf("malformed meminfo line: %s", line)
			}
			kb, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, fmt.Errorf("parse meminfo kB: %w", err)
			}
			return kb / 1024, nil
		}
	}
	return 0, fmt.Errorf("MemAvailable not found in /proc/meminfo")
}

// TierForRAM returns the TierKey for a given MB.
func TierForRAM(mb int) TierKey {
	gb := mb / 1024
	switch {
	case gb < 4:
		return TierNone
	case gb < 6:
		return TierTiny
	case gb < 8:
		return TierSmall
	case gb < 12:
		return TierBalanced
	case gb < 16:
		return TierMedium
	case gb < 24:
		return TierLarge
	case gb < 32:
		return TierXL
	default:
		return TierXXL
	}
}

// RecommendForHost returns the recommended model list for the current host.
// If MemAvailable cannot be read, returns TierNone recommendations.
func RecommendForHost() (TierKey, []ModelRec) {
	mb, err := MemAvailableMB()
	if err != nil {
		return TierNone, nil
	}
	tier := TierForRAM(mb)
	return tier, RecommendationMatrix[DefaultChecksumVersion][tier]
}
