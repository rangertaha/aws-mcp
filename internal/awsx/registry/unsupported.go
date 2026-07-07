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
func unsupported(input, output reflect.Type) (bool, string) {
	if reason := unsupportedStruct(input); reason != "" {
		return true, "input: " + reason
	}
	if reason := unsupportedStruct(output); reason != "" {
		return true, "output: " + reason
	}
	return false, ""
}

// unsupportedStruct checks each top-level field of a struct type.
func unsupportedStruct(t reflect.Type) string {
	if t.Kind() != reflect.Struct {
		return ""
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if reason := unsupportedField(f.Name, f.Type); reason != "" {
			return reason
		}
	}
	return ""
}

// unsupportedField inspects a single field's type, unwrapping pointer,
// slice/array, and map container types to find the element kind that
// actually matters (this is how map[string]types.AttributeValue and
// []types.SomeUnion get caught, without walking into unrelated nested
// struct fields).
func unsupportedField(name string, ft reflect.Type) string {
	switch ft.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		return unsupportedField(name, ft.Elem())
	case reflect.Chan:
		return "field " + name + " is a channel (event-stream)"
	case reflect.Interface:
		if ft.NumMethod() == 0 {
			return "" // empty interface (any): encoding/json handles this natively
		}
		if ft.Implements(jsonMarshalerType) && ft.Implements(jsonUnmarshalerType) {
			return ""
		}
		return "field " + name + " is a non-empty interface type (union/polymorphic shape) without JSON support"
	}
	if ft.Implements(readerType) || ft.Implements(writerType) {
		return "field " + name + " is a streaming io.Reader/io.Writer"
	}
	if ft.Implements(smithyDocumentMarshalerType) || ft.Implements(smithyDocumentUnmarshalerType) {
		return "field " + name + " is an open-content smithy document type"
	}
	return ""
}
