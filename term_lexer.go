package main

import (
	"strings"
)

const (
	itemAtom itemType = iota + itemError + 1
	itemNumber
	itemString
	itemBegList
	itemEndList
	itemBegTuple
	itemEndCurly
	itemBegBinary
	itemEndBinary
	itemBegMap
	itemArrow
	itemComment
	itemComma
	itemDot
	itemEllipsis
)

var singles = map[rune]itemType{
	'{': itemBegTuple,
	'}': itemEndCurly,
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
			return l.errorf("expected binary end")
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
	case r == '#':
		if l.next() != '{' {
			return l.errorf("found # but without #{")
		}
		l.emit(itemBegMap)
		return lexTerm
	case r == '=':
		if l.next() != '>' {
			return l.errorf("found = but without =>")
		}
		l.emit(itemArrow)
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

func lexErlTerm(name, input string) chan item {
	_, items := lex(name, input, lexTerm)
	return items
}
