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
