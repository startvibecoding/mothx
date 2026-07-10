package esm

import "testing"

func TestParseWorkerReportExtractsJSON(t *testing.T) {
	report, err := ParseWorkerReport("```json\n{\"status\":\"complete_candidate\",\"summary\":\"done\",\"evidence\":[\"test passed\"],\"remaining_work\":[],\"blockers\":[]}\n```")
	if err != nil {
		t.Fatalf("ParseWorkerReport: %v", err)
	}
	if report.Status != WorkerStatusCompleteCandidate || report.Summary != "done" || len(report.Evidence) != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestParseWorkerReportAcceptsMissingWorkAlias(t *testing.T) {
	report, err := ParseWorkerReport(`{"status":"continue","summary":"working","missing_work":[" add tests ","  "]}`)
	if err != nil {
		t.Fatalf("ParseWorkerReport: %v", err)
	}
	if len(report.RemainingWork) != 1 || report.RemainingWork[0] != "add tests" {
		t.Fatalf("RemainingWork = %#v", report.RemainingWork)
	}
}

func TestParseWorkerReportMergesAndDeduplicatesRemainingWork(t *testing.T) {
	report, err := ParseWorkerReport(`{"status":"continue","summary":"working","remaining_work":[" implement fix ","run tests"],"missing_work":["implement fix"," update docs ","run tests"]}`)
	if err != nil {
		t.Fatalf("ParseWorkerReport: %v", err)
	}
	want := []string{"implement fix", "run tests", "update docs"}
	if len(report.RemainingWork) != len(want) {
		t.Fatalf("RemainingWork = %#v, want %#v", report.RemainingWork, want)
	}
	for i := range want {
		if report.RemainingWork[i] != want[i] {
			t.Fatalf("RemainingWork = %#v, want %#v", report.RemainingWork, want)
		}
	}
}

func TestParseAuditReportRejectsInvalidVerdict(t *testing.T) {
	if _, err := ParseAuditReport(`{"verdict":"maybe","review":"unclear"}`); err == nil {
		t.Fatal("ParseAuditReport accepted invalid verdict")
	}
}

func TestParseAuditReportPass(t *testing.T) {
	report, err := ParseAuditReport(`{"verdict":"pass","review":"verified","requirements_checked":["req -> ok"],"missing_work":[],"evidence":["go test"]}`)
	if err != nil {
		t.Fatalf("ParseAuditReport: %v", err)
	}
	if report.Verdict != AuditVerdictPass || report.Review != "verified" || len(report.RequirementsChecked) != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestParseRecoveryReport(t *testing.T) {
	report, err := ParseRecoveryReport(`{"decision":"resume","summary":"tests show the partial change is valid","evidence":["go test ./..."],"remaining_work":["finish docs"],"blockers":[]}`)
	if err != nil {
		t.Fatalf("ParseRecoveryReport: %v", err)
	}
	if report.Decision != RecoveryDecisionResume || report.Summary == "" || len(report.RemainingWork) != 1 {
		t.Fatalf("report = %#v", report)
	}
	if _, err := ParseRecoveryReport(`{"decision":"blocked","summary":"cannot continue","blockers":[]}`); err == nil {
		t.Fatal("ParseRecoveryReport accepted blocked report without blocker")
	}
}
