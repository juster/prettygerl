package main

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type itemType int

const (
	itemEOF itemType = iota
	itemError
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
	input io.RuneReader
	ahead rune
	buf   []rune
	line  int
	col   int
	items chan item
}

func (l *lexer) emit(t itemType) {
	var n int
	for _, r := range l.buf {
		m := utf8.RuneLen(r)
		if m < 0 {
			panic(fmt.Errorf("invalid rune: %d", r))
		}
		n += m
	}
	//fmt.Fprintf(os.Stderr, "*DBG* emit(%v) buflen:%d\n", t, len(l.buf))
	b := make([]byte, n)
	var i int
	for _, r := range l.buf {
		i += utf8.EncodeRune(b[i:], r)
	}
	l.items <- item{t, string(b)}
	l.buf = nil
}

// lookAhead gets a fresh lookahead token but avoid appending to our internal buffer.

func (l *lexer) lookAhead() rune {
	r := l.ahead
	if r == eof {
		return eof
	}
	var err error
	l.ahead, _, err = l.input.ReadRune()
	switch err {
	case nil:
		return r
	case io.EOF:
		l.ahead = eof
		return r
	default:
		panic(err)
	}
}

// next returns the next rune in the input.

func (l *lexer) next() rune {
	// push the current lookahead token to the byte buffer
	r := l.lookAhead()
	if r != eof {
		l.putBack(r)
	}
	return r
}

func (l *lexer) putBack(r rune) {
	l.buf = append(l.buf, r)
}

func (l *lexer) backup() rune {
	n := len(l.buf) - 1
	r := l.buf[n]
	l.buf = l.buf[:n]
	return r
}

func (l *lexer) peek() rune {
	return l.ahead
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.buf = nil
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.peek()) < 0 {
		return false
	}
	l.next()
	return true
}

// acceptRun consumes a string of runes from the valid set.
func (l *lexer) acceptRun(valid string) bool {
	var accepted bool
	for l.accept(valid) {
		accepted = true
	}
	return accepted
}

// acceptString consumes a sequence of runes if they match the string literal
func (l *lexer) acceptString(valid string) bool {
	for _, r := range valid {
		if l.next() != r {
			return false
		}
	}
	return true
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *lexer) run(start stateFn) {
	for state := start; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func lex(name string, rdr io.RuneReader, start stateFn) (*lexer, chan item) {
	l := &lexer{
		name:  name,
		input: rdr,
		items: make(chan item),
	}
	// prime the lookahead rune
	l.lookAhead()
	go l.run(start)
	return l, l.items
}
