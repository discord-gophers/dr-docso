package main

import (
	"strings"

	"github.com/hhhapz/doc"
)

func typdef(def string, full bool) (string, bool) {
	split := strings.Split(def, "\n")

	// There is no more to share, this is the full declaration.
	if len(split) == 1 {
		return def, false
	}

	// More than one line, but there is more declaration
	if !full {
		return split[0], true
	}

	b := strings.Builder{}
	b.Grow(len(def))

	var more bool

	for _, line := range strings.Split(def, "\n") {
		b.WriteRune('\n')

		if len(line)+b.Len() > defLimit {
			b.WriteString("// full signature omitted")
			more = true
			break
		}
		b.WriteString(line)
	}
	return b.String(), more
}

func comment(c doc.Comment, initial int, full bool) (string, bool) {
	if len(c) == 0 {
		return "*No documentation found*", false
	}

	if !full {
		if md := c.Markdown(); len(md) < 500 {
			return md, false
		}

		md, more := c[0].Markdown(), len(c) > 1
		if len(md) > 500 {
			md = md[:400]
			more = true
		}
		if more {
			md += "...\n\n*More documentation omitted*"
		}

		return md, more
	}

	var parts doc.Comment
	var more bool

	length := initial
	for _, note := range c {
		l := len(note.Text())
		if l+length > docLimit {
			parts = append(parts, doc.Paragraph("*More documentation omitted...*"))
			more = true
			break
		}
		length += l
		parts = append(parts, note)
	}
	return parts.Markdown(), more
}
