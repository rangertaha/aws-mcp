// SPDX-License-Identifier: MIT

package registry

import "strings"

// readPrefixes are operation-name verb prefixes classified as safe/read-only.
var readPrefixes = []string{
	"Get", "List", "Describe", "Head", "Query", "Scan", "Search",
	"BatchGet", "Lookup", "Check", "Test", "Validate", "Estimate",
	"Simulate", "Preview", "View",
}

// destructivePrefixes are write-operation verb prefixes classified as
// destructive (delete/replace/disable semantics). Only consulted for
// operations already classified as mutating.
var destructivePrefixes = []string{
	"Delete", "Terminate", "Remove", "Deregister", "Revoke", "Reject",
	"Disable", "Stop", "Reboot", "Purge", "Cancel", "Detach", "Disassociate",
}

// classify returns whether an operation is mutating (state-changing) and, if
// so, whether it is specifically destructive, based on a verb-prefix
// heuristic. AWS SDK v2's generated Go clients don't retain smithy HTTP
// traits (e.g. readonly/idempotent) at runtime, so there's no structural
// signal to reflect on; this heuristic is deliberately safe-by-default:
// unrecognized verbs default to Mutating=true. A false positive (hiding a
// genuinely safe operation under read-only mode) is an acceptable cost; a
// false negative (allowing a mutating call through read-only mode) is not.
func classify(operation string) (mutating, destructive bool) {
	for _, p := range readPrefixes {
		if strings.HasPrefix(operation, p) {
			return false, false
		}
	}
	for _, p := range destructivePrefixes {
		if strings.HasPrefix(operation, p) {
			return true, true
		}
	}
	return true, false
}
