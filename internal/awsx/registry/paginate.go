// SPDX-License-Identifier: MIT

package registry

import (
	"reflect"
	"regexp"
)

// paginationFieldPattern matches the field names AWS SDK v2 operations
// conventionally use for continuation tokens/markers.
//
// DynamoDB's family (LastEvaluatedKey/TableName/GlobalTableName/BackupArn/
// StreamArn) is its *output* convention for "pass this back to continue";
// ExclusiveStartKey is the corresponding *input* field name (never an
// output field itself), kept here defensively in case some operation's
// output happens to reuse it. Other "LastEvaluated*" fields exist (e.g.
// ec2's LastEvaluatedTime, ecr's LastEvaluatedAt) but are audit timestamps,
// not tokens meant to be passed back verbatim, so they're deliberately not
// matched here.
var paginationFieldPattern = regexp.MustCompile(`(?i)^(NextToken|NextMarker|Marker|PageToken|ExclusiveStartKey|LastEvaluatedKey|LastEvaluatedTableName|LastEvaluatedGlobalTableName|LastEvaluatedBackupArn|LastEvaluatedStreamArn|ContinuationToken)$`)

// paginationField returns the name of the Output field most likely to carry
// a pagination token/marker, or "" if none matches the common naming
// patterns. AWS SDK v2 also generates separate NewXPaginator constructors for
// many list/describe operations, but those are free functions rather than
// client methods, so they're invisible to (and don't conflict with) the
// reflection-based operation discovery in reflect.go; passing this field's
// value back as the matching Input field on a subsequent aws_invoke call
// achieves the same thing manually.
func paginationField(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		if paginationFieldPattern.MatchString(name) {
			return name
		}
	}
	return ""
}
