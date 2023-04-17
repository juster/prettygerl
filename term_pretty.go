package main

import (
	"fmt"
	"io"
	"strings"
)

const indentOne = "  "

var balanced = map[itemType]itemType{
	itemBegList: itemEndList,
	itemBegTuple: itemEndTuple,
}

type prettier struct {
	out io.Writer
	n int
	spaces string
	printed bool
}

func (p *prettier) newline() {
	fmt.Fprint(p.out, "\n")
	p.printed = false
}

func (p *prettier) indent(dir bool) {
	p.newline()
	if(dir){
		p.n++
	} else if p.n--; p.n < 0 {
		p.n = 0
	}
	//fmt.Printf("*DBG* n=%v\n", p.n)
	p.spaces = strings.Repeat(indentOne, p.n)
}

func (p *prettier) print(val string) {
	if !p.printed {
		fmt.Fprint(p.out, p.spaces)
		p.printed = true
	}
	fmt.Fprint(p.out, val)
}

func prettyErlTerm(in chan item, out io.Writer) error {
	p := &prettier{out: out}
	item := <-in
	nested := make([]itemType, 0, 8)
Loop:
	for {
		peek := <-in
		//fmt.Printf("*DBG* item=%#v -- peek=%#v\n", item, peek)
		switch item.typ {
		case itemEOF:
			break Loop
		case itemError:
			return fmt.Errorf("parse error: %s", item.val)
		case itemEndList, itemEndTuple, itemEndBinary:
			// de-indent before printing closing delimeter
			nested = nested[:len(nested)-1]
			if item.typ != itemEndBinary {
				p.indent(false)
			}
		}

		p.print(item.val)

		switch item.typ {
		case itemComma:
			if nested[len(nested)-1] == itemBegBinary {
				fmt.Fprint(out, " ")
				break
			}
			p.newline()
		case itemBegList, itemBegTuple, itemBegBinary:
			// indent after printing open delimiter, unless it's empty
			if t, ok := balanced[item.typ]; ok && t == peek.typ {
				p.print(peek.val)
				peek = <-in // skip the lookahead
			} else {
				nested = append(nested, item.typ)
				if item.typ != itemBegBinary {
					p.indent(true)
				}
			}
		}
		item = peek
	}
	p.newline()
	return nil
}
