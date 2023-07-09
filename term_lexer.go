package main

import (
	"io"
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
		return lexAtom
	case r == '%':
		return lexComment
	case strings.IndexRune(digits, r) >= 0 || r == '-':
		return lexNumber
	case r == '<':
		if l.next() != '<' {
			l.errorf("expected binary begin")
		}
		l.emit(itemBegBinary)
		return lexTerm
	case r == '>':
		if l.next() != '>' {
			return l.errorf("expected binary end")
		}
		l.emit(itemEndBinary)
		return lexTerm
	case r == '"':
		return lexString
	case r == '.':
		// 1 dot found.
		if l.peek() != '.' {
			l.emit(itemDot)
			return lexTerm
		}
		l.next()

		// 2 dots found.
		if l.peek() != '.' {
			l.emit(itemDot)
			l.emit(itemDot)
			return lexTerm
		}
		l.next()

		// 3 dots found
		l.emit(itemEllipsis)
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
	if tok, ok := singles[r]; ok {
		l.emit(tok)
		return lexTerm
	}
	return l.errorf("unexpected char: %c", r)
}

func lexAtom(l *lexer) stateFn {
	if l.buf[0] == '\'' {
		for {
			r2 := l.next()
			switch r2 {
			case '\'':
				l.emit(itemAtom)
				return lexTerm
			case eof:
				return l.errorf("missing closing '")
			}
		}
	}

Loop:
	for ; ; l.next() {
		r := l.peek()
		switch {
		case (r >= 'a' && r <= 'z') || isDigital(r):
		case r == '_' || r == '@':
		default:
			break Loop
		}
	}
	l.emit(itemAtom)
	return lexTerm
}

func lexNumber(l *lexer) stateFn {
	if l.buf[0] == '-' {
		if !l.acceptRun(digits) {
			return l.errorf("expected number")
		}
	} else {
		l.acceptRun(digits)
	}
	if l.peek() != '.' {
		goto Done
	}
	l.next()
	if !isDigital(l.peek()) {
		// The dot can either represent:
		// 1) a floating point number
		// 2) the end of a term
		// If there is no number after dot, it's the end of a term
		dot := l.backup()
		l.emit(itemNumber)
		l.putBack(dot)
		l.emit(itemDot)
		return lexTerm
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
	for {
		r := l.peek()
		if r == '\n' || r == eof {
			break
		}
		l.next()
	}
	l.emit(itemComment)
	return lexTerm
}

func lexString(l *lexer) stateFn {
	esc := false
Loop:
	for {
		r := l.next()
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

func lexErlTerm(name string, rdr io.RuneReader) chan item {
	_, items := lex(name, rdr, lexTerm)
	return items
}
