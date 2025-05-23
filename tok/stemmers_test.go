/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package tok

import (
	"testing"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/stretchr/testify/require"
)

func TestFilterStemmers(t *testing.T) {
	tests := []struct {
		lang string
		in   analysis.TokenStream
		out  analysis.TokenStream
	}{
		{lang: "en",
			in: analysis.TokenStream{
				&analysis.Token{Term: []byte("the")},
				&analysis.Token{Term: []byte("quick")},
				&analysis.Token{Term: []byte("brown")},
				&analysis.Token{Term: []byte("foxes")},
				&analysis.Token{Term: []byte("jump")},
				&analysis.Token{Term: []byte("over")},
				&analysis.Token{Term: []byte("the")},
				&analysis.Token{Term: []byte("big")},
				&analysis.Token{Term: []byte("dogs")},
			},
			out: analysis.TokenStream{
				&analysis.Token{Term: []byte("the")},
				&analysis.Token{Term: []byte("quick")},
				&analysis.Token{Term: []byte("brown")},
				&analysis.Token{Term: []byte("fox")},
				&analysis.Token{Term: []byte("jump")},
				&analysis.Token{Term: []byte("over")},
				&analysis.Token{Term: []byte("the")},
				&analysis.Token{Term: []byte("big")},
				&analysis.Token{Term: []byte("dog")},
			},
		},
		{lang: "es",
			in: analysis.TokenStream{
				&analysis.Token{Term: []byte("deseándoles")},
				&analysis.Token{Term: []byte("muchas")},
				&analysis.Token{Term: []byte("alegrías")},
				&analysis.Token{Term: []byte("a")},
				&analysis.Token{Term: []byte("las")},
				&analysis.Token{Term: []byte("señoritas")},
				&analysis.Token{Term: []byte("y")},
				&analysis.Token{Term: []byte("los")},
				&analysis.Token{Term: []byte("señores")},
				&analysis.Token{Term: []byte("programadores")},
				&analysis.Token{Term: []byte("de")},
				&analysis.Token{Term: []byte("Dgraph")},
			},
			out: analysis.TokenStream{
				&analysis.Token{Term: []byte("deseándol")},
				&analysis.Token{Term: []byte("much")},
				&analysis.Token{Term: []byte("alegrí")},
				&analysis.Token{Term: []byte("a")},
				&analysis.Token{Term: []byte("las")},
				&analysis.Token{Term: []byte("señorit")},
				&analysis.Token{Term: []byte("y")},
				&analysis.Token{Term: []byte("los")},
				&analysis.Token{Term: []byte("señor")},
				&analysis.Token{Term: []byte("programador")},
				&analysis.Token{Term: []byte("de")},
				&analysis.Token{Term: []byte("Dgraph")},
			},
		},
		{lang: "x-klingon",
			in: analysis.TokenStream{
				&analysis.Token{Term: []byte("tlhIngan")},
				&analysis.Token{Term: []byte("maH!")},
			},
			out: analysis.TokenStream{
				&analysis.Token{Term: []byte("tlhIngan")},
				&analysis.Token{Term: []byte("maH!")},
			},
		},
		{lang: "en",
			in:  analysis.TokenStream{},
			out: analysis.TokenStream{},
		},
		{lang: "",
			in: analysis.TokenStream{
				&analysis.Token{
					Term: []byte(""),
				},
			},
			out: analysis.TokenStream{
				&analysis.Token{
					Term: []byte(""),
				},
			},
		},
	}

	for _, tc := range tests {
		out := filterStemmers(tc.lang, tc.in)
		require.Equal(t, tc.out, out)
	}
}
