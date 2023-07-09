package main

import (
	"fmt"
	"strings"
	"testing"
)

type lexRun struct {
	input  string
	output []item
}

var runs = []lexRun{
	{
		`foo.`,
		[]item{
			{itemAtom, "foo"},
			{itemDot, "."},
			{itemEOF, ""},
		},
	},
	{
		`123.`,
		[]item{
			{itemNumber, "123"},
			{itemDot, "."},
			{itemEOF, ""},
		},
	},
	{
		`123.0.`,
		[]item{
			{itemNumber, `123.0`},
			{itemDot, "."},
			{itemEOF, ""},
		},
	},
	{
		`123.0e+100`,
		[]item{
			{itemNumber, `123.0e+100`},
			{itemEOF, ""},
		},
	},
	{
		`123.0e-100`,
		[]item{
			{itemNumber, `123.0e-100`},
			{itemEOF, ""},
		},
	},
	{
		`123.0e+-100`,
		[]item{
			{itemError, "invalid scientific notation"},
			{itemEOF, ""},
		},
	},
	{
		`"foo"`,
		[]item{
			{itemString, `"foo"`},
			{itemEOF, ""},
		},
	},
	{
		`"foo\t\"bar\"\n"`,
		[]item{
			{itemString, `"foo\t\"bar\"\n"`},
			{itemEOF, ""},
		},
	},
	{
		`'foo+$%^(@#'`,
		[]item{
			{itemAtom, `'foo+$%^(@#'`},
			{itemEOF, ""},
		},
	},
	{
		`[foo, 123.0, "bar\"", ...].`,
		[]item{
			{itemBegList, "["},
			{itemAtom, "foo"},
			{itemComma, ","},
			{itemNumber, "123.0"},
			{itemComma, ","},
			{itemString, `"bar\""`},
			{itemComma, ","},
			{itemEllipsis, "..."},
			{itemEndList, "]"},
			{itemDot, "."},
			{itemEOF, ""},
		},
	},
	{
		`<<"foo", 255, 128, ...>>.`,
		[]item{
			{itemBegBinary, "<<"},
			{itemString, `"foo"`},
			{itemComma, ","},
			{itemNumber, "255"},
			{itemComma, ","},
			{itemNumber, "128"},
			{itemComma, ","},
			{itemEllipsis, "..."},
			{itemEndBinary, ">>"},
			{itemDot, "."},
			{itemEOF, ""},
		},
	},
	{
		`{record, <<"foo">>, 255, 128}.`,
		[]item{
			{itemBegTuple, "{"},
			{itemAtom, "record"},
			{itemComma, ","},
			{itemBegBinary, "<<"},
			{itemString, `"foo"`},
			{itemEndBinary, ">>"},
			{itemComma, ","},
			{itemNumber, "255"},
			{itemComma, ","},
			{itemNumber, "128"},
			{itemEndCurly, "}"},
			{itemDot, "."},
			{itemEOF, ""},
		},
	},
	{
		"[{<<0>>}]. % to be ...\n3133.7. % continued",
		[]item{
			{itemBegList, "["},
			{itemBegTuple, "{"},
			{itemBegBinary, "<<"},
			{itemNumber, "0"},
			{itemEndBinary, ">>"},
			{itemEndCurly, "}"},
			{itemEndList, "]"},
			{itemDot, "."},
			{itemComment, "% to be ..."},
			{itemNumber, "3133.7"},
			{itemDot, "."},
			{itemComment, "% continued"},
			{itemEOF, ""},
		},
	},
}

func TestEOF(t *testing.T) {
	eof_ch := lexErlTerm("eof_test", strings.NewReader(""))
	want := item{itemEOF, ""}
	for i := 0; i < 5; i++ {
		if eof_item := <-eof_ch; eof_item != want {
			t.Errorf("expected EOF as empty value")
		}
	}
}

func TestLexer(t *testing.T) {
	for i, run := range runs {
		t.Run(fmt.Sprintf("Case%d", i), func(t *testing.T) {
			ch := lexErlTerm("lex_test", strings.NewReader(run.input))
			for j, ex := range run.output {
				item := <-ch
				if item != ex {
					t.Errorf("case %d, item %d: expected %#v, got %#v", i, j, ex, item)
				}
			}
		})
	}
}
