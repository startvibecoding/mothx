package esm

import "testing"

func FuzzParseReports(f *testing.F) {
	for _, seed := range []string{
		`{"status":"continue","summary":"working"}`,
		`{"verdict":"pass","review":"verified","requirements_checked":["tests"]}`,
		`{"decision":"resume","summary":"continue work"}`,
		"```json\n{\"status\":\"blocked_candidate\",\"summary\":\"blocked\",\"blockers\":[\"missing access\"]}\n```",
		"{\"status\":\"continue\",\"summary\":\"quote: \\\" and brace: }\"}",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if report, err := ParseWorkerReport(input); err == nil {
			switch report.Status {
			case WorkerStatusContinue, WorkerStatusCompleteCandidate, WorkerStatusBlockedCandidate:
			default:
				t.Fatalf("accepted invalid worker status %q", report.Status)
			}
		}

		if report, err := ParseAuditReport(input); err == nil {
			switch report.Verdict {
			case AuditVerdictPass, AuditVerdictFail:
			default:
				t.Fatalf("accepted invalid audit verdict %q", report.Verdict)
			}
		}

		if report, err := ParseRecoveryReport(input); err == nil {
			switch report.Decision {
			case RecoveryDecisionResume:
				if report.Summary == "" {
					t.Fatal("accepted recovery report without a summary")
				}
			case RecoveryDecisionBlocked:
				if report.Summary == "" || len(report.Blockers) == 0 {
					t.Fatal("accepted blocked recovery report without required fields")
				}
			default:
				t.Fatalf("accepted invalid recovery decision %q", report.Decision)
			}
		}
	})
}
