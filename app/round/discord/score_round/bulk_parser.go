package scoreround

import (
	"fmt"
	"strconv"
	"strings"
)

const minPrefixLen = 3

type bulkUpdate struct {
	UserID string
	Score  int
	Forced bool
	Raw    string
}

type bulkDiagnostics struct {
	line   string
	reason string
}

type nameIndex struct {
	nameToID   map[string]string
	prefixToID map[string]string
}

func buildNameIndex(nameToID map[string]string) nameIndex {
	prefixToID := map[string]string{}
	for ln, uid := range nameToID {
		for l := minPrefixLen; l <= len(ln); l++ {
			p := ln[:l]
			if existing, ok := prefixToID[p]; ok {
				if existing != uid { // collision
					prefixToID[p] = ""
				}
			} else {
				prefixToID[p] = uid
			}
		}
	}
	return nameIndex{nameToID: nameToID, prefixToID: prefixToID}
}

func resolveToken(tok string, idx nameIndex) (string, bool) {
	t := strings.TrimSpace(tok)
	if t == "" {
		return "", false
	}
	t = strings.TrimPrefix(t, "@")
	if t == "" {
		return "", false
	}
	allDigits := true
	for _, r := range t {
		if r < '0' || r > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return t, true
	}
	lt := strings.ToLower(t)
	if id, ok := idx.nameToID[lt]; ok {
		return id, true
	}
	if len(lt) >= minPrefixLen {
		if id, ok := idx.prefixToID[lt]; ok && id != "" {
			return id, true
		}
	}
	return "", false
}

func parseBulkOverrides(raw string, originalScores map[string]int, idx nameIndex) ([]bulkUpdate, int, int, []string, []bulkDiagnostics) {
	lines := strings.Split(raw, "\n")
	var updates []bulkUpdate
	var diagnostics []bulkDiagnostics
	var unresolved []string
	var skipped, unchanged int
	replacer := strings.NewReplacer("<@", "", ">", "", "=", " ", "!", "")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		norm := replacer.Replace(line)
		parts := strings.Fields(norm)
		if len(parts) < 2 {
			skipped++
			diagnostics = append(diagnostics, bulkDiagnostics{line: rawLine, reason: "invalid-format"})
			continue
		}
		token := parts[0]
		participant := token
		resolvedNow := false
		if _, ok := originalScores[participant]; !ok {
			if resolved, ok := resolveToken(token, idx); ok {
				participant = resolved
				resolvedNow = true
			} else {
				unresolved = append(unresolved, token)
				skipped++
				diagnostics = append(diagnostics, bulkDiagnostics{line: rawLine, reason: "unresolved"})
				continue
			}
		}
		scoreStr := parts[1]
		forced := false
		if strings.HasSuffix(scoreStr, "!") {
			forced = true
			scoreStr = strings.TrimSuffix(scoreStr, "!")
		}
		if scoreStr == scoreNoData {
			unchanged++
			diagnostics = append(diagnostics, bulkDiagnostics{line: rawLine, reason: "placeholder"})
			continue
		}
		scoreStr = strings.TrimPrefix(scoreStr, "+")
		scoreVal, err := strconv.Atoi(scoreStr)
		if err != nil || scoreVal < scoreMin || scoreVal > scoreMax {
			skipped++
			diagnostics = append(diagnostics, bulkDiagnostics{line: rawLine, reason: "invalid-score"})
			continue
		}
		if orig, ok := originalScores[participant]; ok && orig == scoreVal && !forced {
			unchanged++
			if resolvedNow {
				diagnostics = append(diagnostics, bulkDiagnostics{line: rawLine, reason: "unchanged(resolved)"})
			} else {
				diagnostics = append(diagnostics, bulkDiagnostics{line: rawLine, reason: "unchanged"})
			}
			continue
		}
		reason := "update"
		if forced {
			reason += "-forced"
		}
		if resolvedNow {
			reason += "(resolved)"
		}
		diagnostics = append(diagnostics, bulkDiagnostics{line: rawLine, reason: reason})
		updates = append(updates, bulkUpdate{UserID: participant, Score: scoreVal, Forced: forced, Raw: rawLine})
	}
	return updates, unchanged, skipped, unresolved, diagnostics
}

func summarizeBulk(updates []bulkUpdate, unchanged, skipped int, unresolved []string, diagnostics []bulkDiagnostics, resolvedMappings []string) string {
	var summary string
	if len(updates) == 0 {
		summary = fmt.Sprintf("No score changes detected. %d unchanged, %d skipped.", unchanged, skipped)
	} else {
		summary = fmt.Sprintf("Bulk override submitted: %d updates, %d unchanged, %d skipped.", len(updates), unchanged, skipped)
	}
	if len(resolvedMappings) > 0 {
		summary += "\nResolved: " + strings.Join(resolvedMappings, ", ")
	}
	if len(unresolved) > 0 {
		summary += "\nUnresolved: " + strings.Join(unresolved, ", ")
	}
	if len(updates) == 0 && len(diagnostics) > 0 {
		limit := len(diagnostics)
		if limit > 25 {
			limit = 25
		}
		summary += "\nDetails:\n"
		for i := 0; i < limit; i++ {
			d := diagnostics[i]
			summary += d.reason + " => " + d.line + "\n"
		}
		if limit < len(diagnostics) {
			summary += "... (truncated)\n"
		}
	}
	return summary
}
