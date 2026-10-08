package limits

import "testing"

func TestEvaluate(t *testing.T) {
	org := Subject{Type: SubjectOrganisation, ID: "org"}
	usage := map[Subject]Usage{
		org: {StatementsTotal: 10, StatementsThisHour: 2, StatementsToday: 5, StorageBytes: 1000},
	}

	if got := evaluate(nil, usage, 1, 10); got != nil {
		t.Fatalf("no rules should pass, got %v", got)
	}

	under := []Rule{{SubjectType: org.Type, SubjectID: org.ID, Metric: MetricStatementsTotal, Max: 11}}
	if got := evaluate(under, usage, 1, 10); got != nil {
		t.Fatalf("exact remaining capacity should pass, got %v", got)
	}

	over := []Rule{{SubjectType: org.Type, SubjectID: org.ID, Metric: MetricStatementsTotal, Max: 10}}
	got := evaluate(over, usage, 1, 10)
	if got == nil || got.Metric != MetricStatementsTotal || got.Current != 10 {
		t.Fatalf("total limit: got %#v", got)
	}

	storage := []Rule{{SubjectType: org.Type, SubjectID: org.ID, Metric: MetricStorageBytes, Max: 1005}}
	got = evaluate(storage, usage, 1, 10)
	if got == nil || got.Metric != MetricStorageBytes {
		t.Fatalf("storage should use bytes, got %#v", got)
	}
	if evaluate(storage, usage, 1, 5) != nil {
		t.Fatalf("storage within remaining bytes should pass")
	}

	hour := []Rule{{SubjectType: org.Type, SubjectID: org.ID, Metric: MetricStatementsPerHour, Max: 2}}
	got = evaluate(hour, usage, 1, 10)
	if got == nil || got.Metric != MetricStatementsPerHour {
		t.Fatalf("hour limit: got %#v", got)
	}
}
