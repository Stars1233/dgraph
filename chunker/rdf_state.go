/*
 * SPDX-FileCopyrightText: © Hypermode Inc. <hello@hypermode.com>
 * SPDX-License-Identifier: Apache-2.0
 */

package chunker

import (
	"strconv"

	"github.com/hypermodeinc/dgraph/v25/lex"
)

// The constants represent different types of lexed Items possible for an rdf N-Quad.
const (
	itemText        lex.ItemType = 5 + iota // plain text
	itemSubject                             // subject, 6
	itemPredicate                           // predicate, 7
	itemObject                              // object, 8
	itemLabel                               // label, 9
	itemLiteral                             // literal, 10
	itemLanguage                            // language, 11
	itemObjectType                          // object type, 12
	itemValidEnd                            // end with dot, 13
	itemComment                             // comment, 14
	itemComma                               // comma, 15
	itemEqual                               // equal, 16
	itemLeftRound                           // '(', 17
	itemRightRound                          // ')', 18
	itemStar                                // *, 19
	itemSubjectFunc                         // uid, 20
	itemObjectFunc                          // uid, 21
	itemVarName                             // 22
)

// These constants keep a track of the depth while parsing an rdf N-Quad.
const (
	atSubject int = iota
	atPredicate
	atObject
	atLabel
	atFacet
)

const (
	lsThan     = '<'
	underscore = '_'
	colon      = ':'
	dash       = '-'
	quote      = '"'
	hash       = '#'
	dot        = '.'
	at         = '@'
	caret      = '^'
	leftRound  = '('
	rightRound = ')'
	comma      = ','
	equal      = '='
)

// This function inspects the next rune and calls the appropriate stateFn.
func lexText(l *lex.Lexer) lex.StateFn {
	for {
		switch r := l.Next(); {
		case r == lsThan || r == underscore:
			if l.Depth == atSubject {
				l.Backup()
				l.Emit(itemText) // emit whatever we have so far.
				return lexSubject
			}

			if l.Depth == atPredicate {
				l.Backup()
				l.Emit(itemText)
				return lexPredicate
			}

			if l.Depth == atObject {
				l.Backup()
				l.Emit(itemText)
				return lexObject
			}

			if l.Depth == atLabel {
				l.Backup()
				l.Emit(itemText)
				return lexLabel
			}

			return l.Errorf("Invalid input: %c at lexText", r)

		case r == quote:
			if l.Depth != atObject {
				return l.Errorf("Invalid quote for non-object.")
			}
			l.Backup()
			l.Emit(itemText)
			return lexObject

		case r == hash:
			if l.Depth != atSubject {
				return l.Errorf("Invalid input: %c at lexText", r)
			}
			return lexComment

		case r == '*':
			l.Depth++
			l.Emit(itemStar)

		case r == leftRound:
			if l.Depth > atObject {
				l.Backup()
				l.Emit(itemText)
				return lexFacets
			}
			return l.Errorf("Invalid input: %c at Facet", r)

		case r == lex.EOF:
			l.Emit(lex.ItemEOF)
			return nil

		case r == dot:
			if l.Depth > atObject {
				l.Emit(itemValidEnd)
				l.Depth = atSubject
			}

		// This should happen when there is either UID or Val function.
		// Hence, we are just checking for u or v
		case r == 'u' || r == 'v':
			if l.Depth != atSubject && l.Depth != atObject {
				return l.Errorf("Unexpected char '%c'", r)
			}
			l.Backup()
			l.Emit(itemText)
			return lexVariable

		case isSpace(r):
			continue

		default:
			l.Errorf("Invalid input: %c at lexText", r)
		}
	}
}

// Assumes that caller has consumed initial '<'
func lexIRIRef(l *lex.Lexer, styp lex.ItemType, sfn lex.StateFn) lex.StateFn {
	if err := lex.IRIRef(l, styp); err != nil {
		return l.Errorf(err.Error())
	}
	return sfn
}

func lexUidNode(l *lex.Lexer, styp lex.ItemType, sfn lex.StateFn) lex.StateFn {
	l.AcceptUntil(isSpace)
	r := l.Peek()
	if r == lex.EOF {
		return l.Errorf("Unexpected end of uid subject")
	}

	in := l.Input[l.Start:l.Pos]
	if _, err := strconv.ParseUint(in, 0, 64); err != nil {
		return l.Errorf("Unable to convert '%v' to UID", in)
	}

	if isSpace(r) {
		l.Emit(styp)
		return sfn
	}

	return l.Errorf("Invalid character '%c' found for UID node itemType: %v", r,
		styp)
}

// Assumes that caller has consumed '_'.
// BLANK_NODE_LABEL ::= '_:' (PN_CHARS_U | [0-9]) ((PN_CHARS | '.')* PN_CHARS)?
func lexBlankNode(l *lex.Lexer, styp lex.ItemType,
	sfn lex.StateFn) lex.StateFn {
	r := l.Next()
	if r != colon {
		return l.Errorf("Invalid character after _. Expected :, found '%c'", r)
	}
	r = l.Next()
	if r == lex.EOF {
		return l.Errorf("Unexpected end of subject")
	}
	if !(isPNCharsU(r) || (r >= '0' && r <= '9')) {
		return l.Errorf("Invalid character in %v after _: , Got '%c'", styp, r)
	}
	lastAccRune, validRune := l.AcceptRun(func(r rune) bool {
		return r == dot || isPNChar(r)
	})
	if validRune && lastAccRune == dot {
		return l.Errorf("Can not end %v with '.'", styp)
	}

	r = l.Peek()
	if r == lex.EOF {
		return l.Errorf("Unexpected end of %v", styp)
	}

	if isSpace(r) {
		l.Emit(styp)
		return sfn
	}

	return l.Errorf("Invalid character '%c' found for itemType: %v", r, styp)
}

func lexSubject(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	// The subject is an IRI, so we lex till we encounter '>'.
	if r == lsThan {
		l.Depth++
		return lexIRIRef(l, itemSubject, lexText)
	}

	// The subject represents a blank node.
	if r == underscore {
		l.Depth++
		return lexBlankNode(l, itemSubject, lexText)
	}
	// See if its an uid
	return lexUidNode(l, itemSubject, lexText)
}

func lexPredicate(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	// The predicate can only be an IRI according to the spec.
	if r != lsThan {
		return l.Errorf("Invalid character in lexPredicate: '%c'", r)
	}

	l.Depth++

	return lexIRIRef(l, itemPredicate, lexText)
}

func lexLanguage(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	if r != at {
		return l.Errorf("Expected @ prefix for lexLanguage")
	}

	l.Ignore()
	r = l.Next()
	if !isLangTagPrefix(r) {
		return l.Errorf("Invalid language tag prefix: '%c'", r)
	}

	lastRune, validRune := l.AcceptRun(isLangTag)
	if validRune && lastRune == dash {
		return l.Errorf("Invalid character - at the end of language literal.")
	}
	l.Emit(itemLanguage)
	return lexText
}

// Assumes '"' has already been encountered.
// literal ::= STRING_LITERAL_QUOTE ('^^' IRIREF | LANGTAG)?
// STRING_LITERAL_QUOTE ::= '"' ([^#x22#x5C#xA#xD] | ECHAR | UCHAR)* '"'
func lexLiteral(l *lex.Lexer) lex.StateFn {
	for {
		r := l.Next()
		if r == '\u005c' { // backslash
			r = l.Next()
			if l.IsEscChar(r) || lex.HasUChars(r, l) || lex.HasXChars(r, l) {
				continue // This would skip over the escaped rune.
			}
			return l.Errorf("Invalid escape character : '%c' in literal", r)
		}

		if r == 0x5c || r == 0xa || r == 0xd { // 0x22 ('"') is endLiteral
			return l.Errorf("Invalid character '%c' in literal.", r)
		}

		if r == lex.EOF || isEndLiteral(r) {
			break
		}
	}

	l.Emit(itemLiteral)
	l.Depth++

	r := l.Peek()
	if r == at {
		return lexLanguage(l)
	}

	if r == caret {
		return lexObjectType(l)
	}

	return lexText
}

func lexObjectType(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	if r != caret {
		return l.Errorf("Expected ^ for lexObjectType")
	}

	r = l.Next()
	if r != caret {
		return l.Errorf("Expected ^^ for lexObjectType")
	}

	l.Ignore()
	r = l.Next()
	if r != lsThan {
		return l.Errorf("Expected < for lexObjectType")
	}

	return lexIRIRef(l, itemObjectType, lexText)
}

func lexObject(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	// The object can be an IRI, blank node, literal.

	if r == lsThan {
		l.Depth++
		return lexIRIRef(l, itemObject, lexText)
	}

	if r == underscore {
		l.Depth++
		return lexBlankNode(l, itemObject, lexText)
	}

	if r == quote {
		return lexLiteral(l)
	}

	return l.Errorf("Invalid char: '%c' at lexObject", r)
}

func lexLabel(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	// Graph label can either be an IRI or a blank node according to spec.
	if r == lsThan {
		l.Depth++
		return lexIRIRef(l, itemLabel, lexText)
	}

	if r == underscore {
		l.Depth++
		return lexBlankNode(l, itemLabel, lexText)
	}
	return l.Errorf("Invalid char: '%c' at lexLabel", r)
}

// lexFacets parses key-value pairs of Facets. sample is :
// ( key1 = "value1", key2=13, key3=, key4 =2.4, key5=2006-01-02T15:04:05,
//
//	key6=2006-01-02 )
func lexFacets(l *lex.Lexer) lex.StateFn {
	r := l.Next()
	if r != leftRound {
		return l.Errorf("Expected '(' but found '%c' at Facet.", r)
	}
	l.Emit(itemLeftRound)

	// we can come here from the lexObject also ;
	// so setting to ahead of atFacet explicitly
	l.Depth = atFacet + 1

forLoop:
	for {
		r = l.Next()
		switch {
		case isSpace(r):
			l.Ignore()
		case r == equal:
			l.Emit(itemEqual)
		case r == comma:
			l.Emit(itemComma)
		case r == rightRound:
			l.Emit(itemRightRound)
			break forLoop
		case r == lex.EOF:
			l.Emit(lex.ItemEOF)
			return nil
		case r == quote:
			if err := l.LexQuotedString(); err != nil {
				return l.Errorf(err.Error())
			}
			l.Emit(itemText)
		default:
			l.AcceptRun(func(r rune) bool {
				return r != equal && !isSpace(r) && r != rightRound && r != comma
			})
			l.Emit(itemText)
		}
	}
	return lexText
}

// lexComment lexes a comment text.
func lexComment(l *lex.Lexer) lex.StateFn {
	l.Backup()
	for {
		r := l.Next()
		if lex.IsEndOfLine(r) || r == lex.EOF {
			break
		}
	}
	l.Emit(itemComment)
	l.Emit(lex.ItemEOF)
	return nil // Stop the run loop.
}

func lexVariable(l *lex.Lexer) lex.StateFn {
	var r rune

	functionName := "uid"
	if r = l.Next(); r == 'v' {
		functionName = "val"
	}
	l.Backup()

	for _, c := range functionName {
		if r = l.Next(); r != c {
			return l.Errorf("Unexpected char '%c' when parsing uid keyword", r)
		}
	}

	if l.Depth == atObject {
		l.Emit(itemObjectFunc)
	} else if l.Depth == atSubject {
		l.Emit(itemSubjectFunc)
	}
	l.IgnoreRun(isSpace)

	if r = l.Next(); r != '(' {
		return l.Errorf("Expected '(' after uid keyword, found: '%c'", r)
	}

	l.Emit(itemLeftRound)
	l.IgnoreRun(isSpace)

	// TODO(Aman): we support all characters in variable names except space and
	// right bracket. we should support only limited characters in variable names.
	// For now, this is fine because variables names must be used once in query
	// block before they can be used here. And, we throw an error if number of
	// used variables are different than number of defined variables.
	acceptVar := func(r rune) bool { return !(isSpace(r) || r == ')') }
	if _, valid := l.AcceptRun(acceptVar); !valid {
		return l.Errorf("Unexpected end of input while reading variable name")
	}
	l.Emit(itemVarName)
	l.IgnoreRun(isSpace)

	if r = l.Next(); r != ')' {
		return l.Errorf("Expected ')' while reading function found: '%c'", r)
	}
	l.Emit(itemRightRound)
	l.Depth++

	return lexText
}

// isSpace returns true if the rune is a tab or space.
func isSpace(r rune) bool {
	return r == '\u0009' || r == '\u0020'
}

func isEndLiteral(r rune) bool {
	return r == quote || r == '\u000d' || r == '\u000a'
}

func isLangTagPrefix(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	default:
		return false
	}
}

// isLangTag returns true if the rune is allowed by the RDF spec.
func isLangTag(r rune) bool {
	if isLangTagPrefix(r) {
		return true
	}

	switch {
	case r == dash:
		return true
	case r >= '0' && r <= '9':
		return true
	default:
		return false
	}
}

// PN_CHARS_BASE ::=   [A-Z] | [a-z] | [#x00C0-#x00D6] | [#x00D8-#x00F6] |
// [#x00F8-#x02FF] | [#x0370-#x037D] | [#x037F-#x1FFF] | [#x200C-#x200D] | [#x2070-#x218F] |
// [#x2C00-#x2FEF] | [#x3001-#xD7FF] | [#xF900-#xFDCF] | [#xFDF0-#xFFFD] | [#x10000-#xEFFFF]
func isPnCharsBase(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
	case r >= 'A' && r <= 'Z':
	case r >= 0xC0 && r <= 0xD6:
	case r >= 0xD8 && r <= 0xF6:
	case r >= 0xF8 && r <= 0x2FF:
	case r >= 0x370 && r <= 0x37D:
	case r >= 0x37F && r <= 0x1FFF:
	case r >= 0x200C && r <= 0x200D:
	case r >= 0x2070 && r <= 0x218F:
	case r >= 0x2C00 && r <= 0x2FEF:
	case r >= 0x3001 && r <= 0xD7FF:
	case r >= 0xF900 && r <= 0xFDCF:
	case r >= 0xFDF0 && r <= 0xFFFD:
	case r >= 0x10000 && r <= 0xEFFFF:
	default:
		return false
	}
	return true
}

// PN_CHARS_U ::= PN_CHARS_BASE | '_' | ':'
func isPNCharsU(r rune) bool {
	return r == underscore || r == colon || isPnCharsBase(r)
}

// PN_CHARS ::= PN_CHARS_U | '-' | [0-9] | #x00B7 | [#x0300-#x036F] | [#x203F-#x2040]
func isPNChar(r rune) bool {
	switch {
	case r == dash:
	case r >= '0' && r <= '9':
	case r == 0xB7:
	case r >= 0x300 && r <= 0x36F:
	case r >= 0x203F && r <= 0x2040:
	default:
		return isPNCharsU(r)
	}
	return true
}
