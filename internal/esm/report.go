package esm

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	WorkerStatusContinue          = "continue"
	WorkerStatusCompleteCandidate = "complete_candidate"
	WorkerStatusBlockedCandidate  = "blocked_candidate"

	AuditVerdictPass = "pass"
	AuditVerdictFail = "fail"

	RecoveryDecisionResume  = "resume"
	RecoveryDecisionBlocked = "blocked"
)

// WorkerReport is the structured final response from an isolated ESM worker.
type WorkerReport struct {
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	Evidence      []string `json:"evidence"`
	RemainingWork []string `json:"remaining_work"`
	Blockers      []string `json:"blockers"`
}

// AuditReport is the structured final response from an isolated ESM auditor.
type AuditReport struct {
	Verdict             string   `json:"verdict"`
	Review              string   `json:"review"`
	RequirementsChecked []string `json:"requirements_checked"`
	MissingWork         []string `json:"missing_work"`
	Evidence            []string `json:"evidence"`
}

// RecoveryReport is the structured result of an observer inspecting work left
// behind by an interrupted ESM role.
type RecoveryReport struct {
	Decision      string   `json:"decision"`
	Summary       string   `json:"summary"`
	Evidence      []string `json:"evidence"`
	RemainingWork []string `json:"remaining_work"`
	Blockers      []string `json:"blockers"`
}

func ParseWorkerReport(text string) (WorkerReport, error) {
	var payload struct {
		WorkerReport
		MissingWork []string `json:"missing_work"`
	}
	if err := decodeReport(text, &payload); err != nil {
		return payload.WorkerReport, err
	}
	report := payload.WorkerReport
	report.Status = strings.TrimSpace(report.Status)
	switch report.Status {
	case WorkerStatusContinue, WorkerStatusCompleteCandidate, WorkerStatusBlockedCandidate:
	default:
		return report, fmt.Errorf("invalid worker status %q", report.Status)
	}
	report.Summary = strings.TrimSpace(report.Summary)
	report.Evidence = trimStringSlice(report.Evidence)
	report.RemainingWork = mergeTrimmedStringSlices(report.RemainingWork, payload.MissingWork)
	report.Blockers = trimStringSlice(report.Blockers)
	return report, nil
}

func mergeTrimmedStringSlices(slices ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, values := range slices {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}

func ParseAuditReport(text string) (AuditReport, error) {
	var report AuditReport
	if err := decodeReport(text, &report); err != nil {
		return report, err
	}
	report.Verdict = strings.TrimSpace(report.Verdict)
	switch report.Verdict {
	case AuditVerdictPass, AuditVerdictFail:
	default:
		return report, fmt.Errorf("invalid audit verdict %q", report.Verdict)
	}
	report.Review = strings.TrimSpace(report.Review)
	report.RequirementsChecked = trimStringSlice(report.RequirementsChecked)
	report.MissingWork = trimStringSlice(report.MissingWork)
	report.Evidence = trimStringSlice(report.Evidence)
	return report, nil
}

func ParseRecoveryReport(text string) (RecoveryReport, error) {
	var report RecoveryReport
	if err := decodeReport(text, &report); err != nil {
		return report, err
	}
	report.Decision = strings.TrimSpace(report.Decision)
	switch report.Decision {
	case RecoveryDecisionResume, RecoveryDecisionBlocked:
	default:
		return report, fmt.Errorf("invalid recovery decision %q", report.Decision)
	}
	report.Summary = strings.TrimSpace(report.Summary)
	report.Evidence = trimStringSlice(report.Evidence)
	report.RemainingWork = trimStringSlice(report.RemainingWork)
	report.Blockers = trimStringSlice(report.Blockers)
	if report.Summary == "" {
		return report, fmt.Errorf("recovery summary is empty")
	}
	if report.Decision == RecoveryDecisionBlocked && len(report.Blockers) == 0 {
		return report, fmt.Errorf("blocked recovery report has no concrete blocker")
	}
	return report, nil
}

func decodeReport(text string, v any) error {
	payload, err := extractJSONObject(text)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(strings.NewReader(payload))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("parse report json: %w", err)
	}
	return nil
}

func extractJSONObject(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("empty report")
	}
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return "", fmt.Errorf("report does not contain a json object")
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1], nil
			}
		}
	}
	return "", fmt.Errorf("unterminated json object")
}

func trimStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
