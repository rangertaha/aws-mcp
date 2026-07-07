package registry

import "testing"

func TestSanityCatalog(t *testing.T) {
	cat := Build(Factories)
	total := 0
	for _, name := range cat.ServiceNames() {
		svc, _ := cat.Service(name)
		total += len(svc.Operations)
	}
	t.Logf("total services=%d total ops=%d", len(cat.Services), total)

	checks := map[string][]string{
		"s3":       {"ListBuckets", "PutObject", "GetObject"},
		"dynamodb": {"PutItem", "Query", "GetItem"},
		"ec2":      {"DescribeInstances", "RunInstances", "TerminateInstances"},
		"lambda":   {"ListFunctions", "Invoke"},
	}
	for svcName, ops := range checks {
		svc, ok := cat.Service(svcName)
		if !ok {
			t.Fatalf("missing service %s", svcName)
		}
		t.Logf("%s: %d operations", svcName, len(svc.Operations))
		for _, opName := range ops {
			op, ok := svc.Operations[opName]
			if !ok {
				t.Fatalf("missing operation %s.%s", svcName, opName)
			}
			t.Logf("  %-20s mutating=%v destructive=%v unsupported=%v reason=%q pagination=%q",
				opName, op.Mutating, op.Destructive, op.Unsupported, op.UnsupportedReason, op.PaginationField)
		}
	}
}
