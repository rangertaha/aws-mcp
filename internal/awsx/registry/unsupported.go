// SPDX-License-Identifier: MIT

package registry

import (
	"encoding/json"
	"io"
	"reflect"
)

var (
	readerType = reflect.TypeFor[io.Reader]()
	writerType = reflect.TypeFor[io.Writer]()

	jsonMarshalerType   = reflect.TypeFor[json.Marshaler]()
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
)

// smithyDocumentMarshalerType/smithyDocumentUnmarshalerType mirror
// smithy-go's document.Marshaler/Unmarshaler interfaces structurally (by
// method set) so detecting them doesn't require an explicit dependency on
// smithy-go's document package.
var (
	smithyDocumentMarshalerType   = reflect.TypeOf((*interface{ MarshalSmithyDocument() ([]byte, error) })(nil)).Elem()
	smithyDocumentUnmarshalerType = reflect.TypeOf((*interface{ UnmarshalSmithyDocument([]byte) error })(nil)).Elem()
)

// unsupported inspects an operation's Input/Output struct types for shapes
// that generic JSON-based dispatch can't handle safely — streaming bodies
// (e.g. s3.PutObjectInput.Body io.Reader), open-content "document" fields
// (e.g. EventBridge PutEvents Detail), and non-empty interface/union types
// encoding/json can't marshal or unmarshal generically (e.g.
// dynamodb/types.AttributeValue). It returns whether the operation should be
// marked Unsupported and, if so, a human-readable reason.
//
// Input and output are checked asymmetrically for non-empty interfaces:
// json.Unmarshal can never populate one (there's no type tag in JSON telling
// it which concrete type to allocate), so any such field anywhere in the
// input — including nested inside another struct, e.g.
// dynamodb.TransactWriteItemsInput's TransactItems[].Put.Item — makes the
// operation impossible to call generically. json.Marshal, by contrast,
// always follows an interface field's actual concrete value regardless of
// its static type, so a populated union/interface field in the output
// marshals just fine; only genuinely unmarshalable *output* shapes
// (streaming bodies, open-content documents, channels) make an operation
// unsupported on the output side.
func unsupported(input, output reflect.Type) (bool, string) {
	if reason := unsupportedStruct(input, true, map[reflect.Type]bool{}); reason != "" {
		return true, "input: " + reason
	}
	if reason := unsupportedStruct(output, false, map[reflect.Type]bool{}); reason != "" {
		return true, "output: " + reason
	}
	return false, ""
}

// unsupportedStruct checks every exported field of a struct type, recursing
// into nested structs (guarded by visited against the mutually- and
// self-referential types common in AWS SDK models, e.g. IAM policy/step
// function definitions). Since the check is purely a function of a type —
// never a particular value — a type only needs to be walked once: if it
// comes up clean here, it's clean everywhere it's reachable from.
func unsupportedStruct(t reflect.Type, forInput bool, visited map[reflect.Type]bool) string {
	if t.Kind() != reflect.Struct {
		return ""
	}
	if visited[t] {
		return ""
	}
	visited[t] = true

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// Deliberately not filtered to exported-only fields: AWS SDK v2's
		// event-stream operations (e.g. bedrockruntime's
		// InvokeModelWithBidirectionalStream) store their genuinely
		// undispatchable channel plumbing in unexported fields
		// (eventStream, initialReply, ...) while still using it as the
		// actual mechanism, so skipping unexported fields would silently
		// mark those operations as safe to invoke.
		if reason := unsupportedField(f.Name, f.Type, forInput, visited); reason != "" {
			return reason
		}
	}
	return ""
}

// unsupportedField inspects a single field's type, unwrapping pointer,
// slice/array, and map container types (this is how map[string]types.AttributeValue
// and []types.SomeUnion get caught) and recursing into nested structs via
// unsupportedStruct.
func unsupportedField(name string, ft reflect.Type, forInput bool, visited map[reflect.Type]bool) string {
	switch ft.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return unsupportedField(name, ft.Elem(), forInput, visited)
	case reflect.Chan:
		return "field " + name + " is a channel (event-stream)"
	case reflect.Interface:
		if ft.NumMethod() == 0 {
			return "" // empty interface (any): encoding/json handles this natively
		}
		// Streaming interfaces (io.Reader/io.ReadCloser on outputs,
		// io.Reader on inputs) are declared as interface-kind fields in AWS
		// SDK v2, so this must be checked before the input/output asymmetry
		// below — unlike a populated union value, a stream can't be
		// marshaled safely in either direction.
		if ft.Implements(readerType) || ft.Implements(writerType) {
			return "field " + name + " is a streaming io.Reader/io.Writer"
		}
		if ft.Implements(smithyDocumentMarshalerType) || ft.Implements(smithyDocumentUnmarshalerType) {
			return "field " + name + " is an open-content smithy document type"
		}
		if ft.Implements(jsonMarshalerType) && ft.Implements(jsonUnmarshalerType) {
			return ""
		}
		if !forInput {
			// A populated union/interface field marshals fine via its
			// concrete runtime value even without custom (Un)MarshalJSON;
			// only decoding (input) genuinely can't handle these.
			return ""
		}
		return "field " + name + " is a non-empty interface type (union/polymorphic shape) without JSON support"
	case reflect.Struct:
		return unsupportedStruct(ft, forInput, visited)
	}
	if ft.Implements(readerType) || ft.Implements(writerType) {
		return "field " + name + " is a streaming io.Reader/io.Writer"
	}
	if ft.Implements(smithyDocumentMarshalerType) || ft.Implements(smithyDocumentUnmarshalerType) {
		return "field " + name + " is an open-content smithy document type"
	}
	return ""
}
