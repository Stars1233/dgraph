/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package schema

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dgraph-io/gqlparser/v2/gqlerror"
	"github.com/hypermodeinc/dgraph/v25/x"
)

func TestDataAndErrors(t *testing.T) {

	tests := map[string]struct {
		data     []string
		errors   []error
		expected string
	}{
		"empty response": {
			data:     nil,
			errors:   nil,
			expected: `{}`,
		},
		"add initial": {
			data:     []string{`{"Some": "Data"}`},
			errors:   nil,
			expected: `{"data": {"Some": "Data"}}`,
		},
		"add nothing": {
			data:     []string{`{"Some": "Data"}`, ""},
			errors:   nil,
			expected: `{"data": {"Some": "Data"}}`,
		},
		"add more": {
			data:     []string{`{"Some": "Data"}`, `{"And": "More"}`},
			errors:   nil,
			expected: `{"data": {"Some": "Data", "And": "More"}}`,
		},
		"errors and data": {
			data:   []string{`{"Some": "Data"}`, `{"And": "More"}`},
			errors: []error{errors.New("An Error")},
			expected: `{
				"errors":[{"message":"An Error"}],
				"data": {"Some": "Data", "And": "More"}}`,
		},
		"many errors": {
			data:   []string{`{"Some": "Data"}`},
			errors: []error{errors.New("An Error"), errors.New("Another Error")},
			expected: `{
				"errors":[{"message":"An Error"}, {"message":"Another Error"}],
				"data": {"Some": "Data"}}`,
		},
		"gql error": {
			data: []string{`{"Some": "Data"}`},
			errors: []error{
				&x.GqlError{Message: "An Error", Locations: []x.Location{{Line: 1, Column: 1}}}},
			expected: `{
				"errors":[{"message":"An Error", "locations": [{"line":1,"column":1}]}],
				"data": {"Some": "Data"}}`,
		},
		"gql error with path": {
			data: []string{`{"Some": "Data"}`},
			errors: []error{
				&x.GqlError{
					Message:   "An Error",
					Locations: []x.Location{{Line: 1, Column: 1}},
					Path:      []interface{}{"q", 2, "n"}}},
			expected: `{
				"errors":[{
					"message":"An Error",
					"locations": [{"line":1,"column":1}],
					"path": ["q", 2, "n"]}],
				"data": {"Some": "Data"}}`,
		},
		"gql error list": {
			data: []string{`{"Some": "Data"}`},
			errors: []error{x.GqlErrorList{
				&x.GqlError{Message: "An Error", Locations: []x.Location{{Line: 1, Column: 1}}},
				&x.GqlError{Message: "Another Error", Locations: []x.Location{{Line: 1, Column: 1}}}}},
			expected: `{
				"errors":[
					{"message":"An Error", "locations": [{"line":1,"column":1}]},
					{"message":"Another Error", "locations": [{"line":1,"column":1}]}],
				"data": {"Some": "Data"}}`,
		},
	}

	for name, tcase := range tests {
		t.Run(name, func(t *testing.T) {
			resp := &Response{}

			for _, d := range tcase.data {
				resp.AddData([]byte(d))
			}
			for _, e := range tcase.errors {
				resp.WithError(e)
			}

			buf := new(bytes.Buffer)
			_, err := resp.WriteTo(buf)
			require.NoError(t, err)

			assert.JSONEq(t, tcase.expected, buf.String())
		})
	}
}

func TestWriteTo_BadData(t *testing.T) {
	resp := &Response{}
	resp.AddData([]byte(`not json`))

	buf := new(bytes.Buffer)
	_, err := resp.WriteTo(buf)
	require.NoError(t, err)

	assert.JSONEq(t,
		`{"errors":[{"message":"Internal error - failed to marshal a valid JSON response"}],
		"data": null}`,
		buf.String())
}

func TestErrorResponse(t *testing.T) {

	tests := map[string]struct {
		err      error
		expected string
	}{
		"an error": {
			err:      errors.New("An Error"),
			expected: `{"errors":[{"message":"An Error"}]}`,
		},

		"an x.GqlError": {
			err: x.GqlErrorf("A GraphQL error").
				WithLocations(x.Location{Line: 1, Column: 2}),
			expected: `
			{"errors":[{"message": "A GraphQL error", "locations": [{"column":2, "line":1}]}]}`},
		"an x.GqlErrorList": {
			err: x.GqlErrorList{
				x.GqlErrorf("A GraphQL error"),
				x.GqlErrorf("Another GraphQL error").WithLocations(x.Location{Line: 1, Column: 2})},
			expected: `{"errors":[
				{"message":"A GraphQL error"},
				{"message":"Another GraphQL error", "locations": [{"column":2, "line":1}]}]}`},
		"a gqlerror": {
			err: &gqlerror.Error{
				Message:   "A GraphQL error",
				Locations: []gqlerror.Location{{Line: 1, Column: 2}}},
			expected: `{
				"errors":[{"message":"A GraphQL error", "locations": [{"line":1,"column":2}]}]}`,
		},
		"a list of gql errors": {
			err: gqlerror.List{
				gqlerror.Errorf("A GraphQL error"),
				&gqlerror.Error{
					Message:   "Another GraphQL error",
					Locations: []gqlerror.Location{{Line: 1, Column: 2}}}},
			expected: `{"errors":[
				{"message":"A GraphQL error"},
				{"message":"Another GraphQL error", "locations": [{"line":1,"column":2}]}]}`,
		},
	}

	for name, tcase := range tests {
		t.Run(name, func(t *testing.T) {

			// ErrorResponse doesn't add data - it should only be called before starting
			// execution - so in all cases no data should be present.
			resp := ErrorResponse(tcase.err)

			buf := new(bytes.Buffer)
			_, err := resp.WriteTo(buf)
			require.NoError(t, err)

			assert.JSONEq(t, tcase.expected, buf.String())
		})
	}
}

func TestNilResponse(t *testing.T) {
	var resp *Response

	buf := new(bytes.Buffer)
	_, err := resp.WriteTo(buf)
	require.NoError(t, err)

	assert.JSONEq(t,
		`{"errors":[{"message":"Internal error - no response to write."}],
		"data": null}`,
		buf.String())
}
