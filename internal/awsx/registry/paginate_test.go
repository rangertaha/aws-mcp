// SPDX-License-Identifier: MIT

package registry

import "testing"

// TestPaginationFieldDynamoDB pins down DynamoDB's pagination field names:
// Query/Scan use LastEvaluatedKey (an output field), not ExclusiveStartKey
// (the corresponding *input* field name for the next page, which never
// appears as an output field and so must never be what paginationField
// matches against for these operations).
func TestPaginationFieldDynamoDB(t *testing.T) {
	cat := Build(Factories)

	checks := map[string]string{
		"Query":            "LastEvaluatedKey",
		"Scan":             "LastEvaluatedKey",
		"ListTables":       "LastEvaluatedTableName",
		"ListGlobalTables": "LastEvaluatedGlobalTableName",
		"ListBackups":      "LastEvaluatedBackupArn",
	}
	for op, want := range checks {
		spec, ok := cat.Operation("dynamodb", op)
		if !ok {
			t.Fatalf("missing dynamodb.%s", op)
		}
		if spec.PaginationField != want {
			t.Errorf("dynamodb.%s: PaginationField = %q, want %q", op, spec.PaginationField, want)
		}
	}
}
