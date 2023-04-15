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

func (l *lexer) run(start stateFn) {
	for state := start; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func lex(name, input string, start stateFn) (*lexer, chan item) {
	l := &lexer{
		line:  1,
		col:   1,
		name:  name,
		input: input,
		items: make(chan item),
	}
	go l.run(start)
	return l, l.items
}
