package spec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Spec struct {
	Nodes    []*Node
	keywords map[string]map[*Node]struct{}
	Keywords map[string][]*Node
}

func (s Spec) Search(query string) []*Node {
	query = strings.ToLower(query)
	fields := strings.Fields(query)

	switch len(fields) {
	case 0:
		return nil
	}

	results := map[*Node]int{}

	for _, f := range fields {
		for _, node := range s.Keywords[f] {

			// exact match
			if strings.ToLower(node.Heading) == query {
				return []*Node{node}
			}

			results[node]++
		}
	}

	keys := make([]*Node, 0, len(results))
	for n, num := range results {
		if num == len(fields) {
			keys = append(keys, n)
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Level == keys[j].Level {
			return keys[i].Heading < keys[j].Heading
		}
		return keys[i].Level < keys[j].Level
	})
	return keys
}

type Node struct {
	Level   int
	Heading string
	Content []Note
	Nodes   []*Node
}

type Note interface {
	Markdown() string
}

type Heading struct {
	Level int
	Text  string
}

type Link struct {
	Text     string
	Location string
}

type List struct {
	Ordered bool
	Items   []Paragraph
}

type (
	Paragraph []Note
	Code      string
	Italic    string
	Pre       string
	Text      string
)

func (n Node) Markdown(limit int) (string, bool) {
	switch len(n.Content) {
	case 0:
		return "", false
	}

	var more bool

	var b strings.Builder
	for _, c := range n.Content {
		if b.Len() > limit {
			more = true
			break
		}
		b.WriteString(c.Markdown())
		b.WriteRune('\n')
	}

	for _, node := range n.Nodes {
		if b.Len() > limit {
			more = true
			break
		}
		md, _ := node.Markdown(limit - b.Len())
		b.WriteString(md)
		b.WriteRune('\n')
	}

	return b.String(), more
}

func (h Heading) Markdown() string {
	switch h.Level {
	case 2:
		return "> __**" + h.Text + "**__\n"
	case 3:
		return "> __" + h.Text + "__\n"
	case 4:
		return "> " + h.Text + "\n"
	}
	return h.Text
}

func (p Paragraph) Markdown() string {
	switch len(p) {
	case 0:
		return ""
	case 1:
		return p[0].Markdown()
	}

	var b strings.Builder
	for _, part := range p {
		b.WriteString(part.Markdown())
	}

	return strings.TrimSpace(b.String()) + "\n"
}

func (t Text) Markdown() string {
	return string(t)
}

func (t Link) Markdown() string {
	return fmt.Sprintf("[%s](%s)", string(t.Text), t.Location)
}

const bullet = "  • "

func (l List) prefix(n int) string {
	if l.Ordered {
		return strconv.Itoa(n) + "  . "
	}
	return bullet
}

func (l List) Markdown() string {
	switch len(l.Items) {
	case 0:
		return ""
	case 1:
		return l.prefix(1) + l.Items[0].Markdown()
	}

	var b strings.Builder

	b.WriteString(l.prefix(1))
	b.WriteString(l.Items[0].Markdown())

	i := 2
	for _, n := range l.Items[1:] {
		b.WriteRune('\n')
		b.WriteString(l.prefix(i))
		b.WriteString(n.Markdown())
		i++
	}

	return b.String()
}

func (c Code) Markdown() string {
	return "`" + string(c) + "`"
}

func (i Italic) Markdown() string {
	return "*" + string(i) + "*"
}

func (p Pre) Markdown() string {
	return "```\n" + string(p) + "\n```\n"
}
