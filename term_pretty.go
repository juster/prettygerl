package main

import (
	"fmt"
	"io"
	"strings"
)

const indentOne = "  "

var balanced = map[itemType]itemType{
	itemBegList:  itemEndList,
	itemBegTuple: itemEndCurly,
	itemBegMap:   itemEndCurly,
}

type indenter struct {
	out       io.Writer
	n         int
	spaces    string
	indented  bool
	dirtyline bool
}

func (p *indenter) newline() {
	if !p.dirtyline {
		return
	}
	fmt.Fprint(p.out, "\n")
	p.indented = false
	p.dirtyline = false
}

func (p *indenter) indent(dir bool) {
	p.newline()
	if dir {
		p.n++
	} else if p.n--; p.n < 0 {
		p.n = 0
	}
	p.spaces = strings.Repeat(indentOne, p.n)
}

func (p *indenter) print(val string) {
	if !p.indented {
		fmt.Fprint(p.out, p.spaces)
		p.indented = true
	}
	fmt.Fprint(p.out, val)
	p.dirtyline = true
}

func prettyErlTerm(in chan item, out io.Writer) error {
	p := &indenter{out: out}
	stack := make([]itemType, 0, 8)
	item := <-in

Loop:
	for {
		//fmt.Printf("*DBG* item=%#v\n", item)
		switch item.typ {
		case itemEOF:
			break Loop
		case itemError:
			return fmt.Errorf("parse error: %s", item.val)
		case itemEndList, itemEndCurly, itemEndBinary:
			// de-indent before printing closing delimeter
			switch {
			case item.typ == itemEndBinary:
			case isPropList(stack):
				// ... except these since we didn't indent them
			default:
				p.indent(false)
			}
			stack = pop(stack)
		case itemArrow:
			p.print(" ")
		}

		p.print(item.val)

		switch item.typ {
		case itemDot, itemComment:
			p.newline()
		case itemComma:
			switch {
			case topEquals(stack, itemBegBinary) || isPropList(stack):
				fmt.Fprint(out, " ")
			default:
				p.newline()
			}
		case itemArrow:
			p.print(" ")
		case itemBegList, itemBegTuple, itemBegBinary, itemBegMap:
			// indent after printing open delimiter, unless it's empty
			peek := <-in
			//fmt.Printf("*DBG* item=%#v -- peek=%#v\n", item, peek)
			if t, ok := balanced[item.typ]; ok && t == peek.typ {
				p.print(peek.val)
				break
			}
			switch {
			case item.typ == itemBegTuple && topEquals(stack, itemBegList):
				// avoids indenting tuples in property lists on their same line
			case item.typ == itemBegBinary:
				// avoids indenting binaries, they're balanced but not nested
			default:
				p.indent(true)
			}
			stack = push(stack, item.typ)

			item = peek
			continue Loop
		}

		item = <-in
	}
	//p.newline()
	return nil
}

func push(stack []itemType, new itemType) []itemType {
	return append([]itemType{new}, stack...)
}

func pop(stack []itemType) []itemType {
	if len(stack) == 0 {
		return stack
	}
	return stack[1:]
}

func isPropList(stack []itemType) bool {
	return topEquals(stack, itemBegTuple, itemBegList)
}

func topEquals(stack []itemType, want ...itemType) bool {
	if len(stack) < len(want) {
		return false
	}
	for i, t := range want {
		if stack[i] != t {
			return false
		}
	}
	return true
}
