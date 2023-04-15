package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type itemType int

const (
	itemEOF itemType = iota
	itemError
	itemAtom
	itemNumber
	itemString
	itemBinary
	itemBegList
	itemEndList
	itemBegTuple
	itemEndTuple
	itemBegBinary
	itemEndBinary
	itemComment
	itemComma
	itemDot
	itemEllipsis
)

const eof = -1

type item struct {
	typ itemType
	val string
}

func (i item) String() string {
	return i.val
}

type stateFn func(*lexer) stateFn

type lexer struct {
	name  string
	input string
	start int
	pos   int
	width int
	line  int
	col   int
	items chan item
}

func (l *lexer) run() {
	for state := lexTerm; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

// next returns the next rune in the input.
func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width =
		utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	if r == '\n' {
		l.line++
		l.col = 1
	}
	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune.
func (l *lexer) backup() {
	if l.pos > 0 && l.input[l.pos-1] == '\n' {
		l.line--
	}
	l.pos -= l.width
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.input) {
		return eof
	}
	r, _ :=
		utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a string of runes from the valid set.
func (l *lexer) acceptRun(valid string) bool {
	seen := false
	for strings.IndexRune(valid, l.next()) >= 0 {
		seen = true
	}
	l.backup()
	return seen
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func lex(name, input string) (*lexer, chan item) {
	l := &lexer{
		line:  1,
		col:   1,
		name:  name,
		input: input,
		items: make(chan item),
	}
	go l.run()
	return l, l.items
}

var singles = map[rune]itemType{
	'{': itemBegTuple,
	'}': itemEndTuple,
	'[': itemBegList,
	']': itemEndList,
	',': itemComma,
}

const digits = "1234567890"

func isDigital(r rune) bool {
	return (strings.IndexRune(digits, r) >= 0)
}

func lexTerm(l *lexer) stateFn {
	var r rune
	switch r = l.next(); {
	case r == eof:
		l.emit(itemEOF)
		return nil
	case strings.IndexRune(" \t\n", r) >= 0:
		l.ignore()
		return lexTerm
	case r == '\'' || r >= 'a' && r <= 'z':
		l.backup()
		return lexAtom
	case r == '%':
		return lexComment
	case strings.IndexRune(digits, r) >= 0 || r == '-':
		l.backup()
		return lexNumber
	case r == '<':
		if r = l.next(); r != '<' {
			l.errorf("expected binary begin")
		}
		l.emit(itemBegBinary)
		return lexTerm
	case r == '>':
		if r = l.next(); r != '>' {
			l.errorf("expected binary end")
		}
		l.emit(itemEndBinary)
		return lexTerm
	case r == '"':
		return lexString
	case r == '.':
		if l.next() == '.' && l.peek() == '.' {
			l.next()
			l.emit(itemEllipsis)
			return lexTerm
		}
		l.backup()
		l.emit(itemDot)
		return lexTerm
	}
	if t, ok := singles[r]; ok {
		l.emit(t)
		return lexTerm
	}
	return l.errorf("unexpected char: %c", r)
}

func lexAtom(l *lexer) stateFn {
	r := l.next()
	if r == '\'' {
		i := strings.IndexRune(l.input[l.pos:], '\'')
		if i < 0 {
			return l.errorf("missing closing '")
		}
		l.pos += i + 1
		l.emit(itemAtom)
		return lexTerm
	}

	for ; ; r = l.next() {
		switch {
		case (r >= 'a' && r <= 'z') || isDigital(r):
			continue
		case r == '_' || r == '@':
			continue
		}
		l.backup()
		if l.pos <= l.start {
			return l.errorf("invalid atom")
		}
		break
	}
	l.emit(itemAtom)
	return lexTerm
}

func lexNumber(l *lexer) stateFn {
	l.accept("-")
	if !l.acceptRun(digits) {
		return l.errorf("expected number")
	}
	if l.peek() != '.' {
		goto Done
	}
	l.next()
	if !isDigital(l.peek()) {
		// The dot can either represent:
		// 1) a floating point number
		// 2) the end of a term
		// If there is no number, putback the dot.
		l.backup()
		goto Done
	}
	l.acceptRun(digits)
	if !l.accept("e") {
		goto Done
	}
	l.accept("-+")
	if !l.acceptRun(digits) {
		return l.errorf("invalid scientific notation")
	}

Done:
	l.emit(itemNumber)
	return lexTerm
}

func lexComment(l *lexer) stateFn {
	if i := strings.IndexRune(l.input[l.pos:], '\n'); i < 0 {
		l.pos = len(l.input)
	} else {
		l.pos += i
	}
	l.emit(itemComment)
	return lexTerm
}

func lexString(l *lexer) stateFn {
	r := l.next()
	esc := false
Loop:
	for ; ; r = l.next() {
		switch r {
		case '\\':
			esc = true
			continue
		case '"':
			if !esc {
				break Loop
			}
		case eof:
			return l.errorf("missing closing \"")
		}
		esc = false
	}
	l.emit(itemString)
	return lexTerm
}
