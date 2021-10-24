package spec

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

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
		n1 := keys[i]
		n2 := keys[j]

		var basic bool
		if n1.Level != n2.Level {
			// show lower matches first - more specific
			basic = n2.Level < n1.Level
		} else {
			basic = n1.Heading < n2.Heading
		}

		title1, title2 := strings.ToLower(n1.Heading), strings.ToLower(n2.Heading)
		c1, c2 := strings.Contains(title1, query), strings.Contains(title2, query)
		switch {
		case c1 && c2:
			return basic
		case c1:
			return true
		case c2:
			return false
		}

		var desc1, desc2 strings.Builder
		for _, d := range n1.Content {
			desc1.WriteString(strings.ToLower(d.Markdown()))
		}
		for _, d := range n2.Content {
			desc2.WriteString(strings.ToLower(d.Markdown()))
		}

		c1, c2 = strings.Contains(desc1.String(), query), strings.Contains(desc2.String(), query)
		switch {
		case c1 && c2:
			return basic
		case c1:
			return true
		case c2:
			return false
		}

		return basic
	})
	return keys
}

func (n Node) Match() string {
	return fmt.Sprintf("> [%s](%s#%s)\n", n.Heading, page, strings.ReplaceAll(n.Heading, " ", "_"))
}

func (n Node) Render(limit int) (string, bool) {
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
		md, _ := node.Render(limit - b.Len())
		b.WriteString(md)
		b.WriteRune('\n')
	}

	if more {
		b.WriteString("*More documentation omitted*")
	}

	return b.String(), more
}

func (h Heading) Markdown() string {
	var text string
	switch h.Level {
	case 2:
		text = "__**" + h.Text + "**__"
	case 3:
		text = "__" + h.Text + "__"
	case 4:
		text = h.Text
	}
	return fmt.Sprintf("> [%s](%s#%s)\n", text, page, strings.ReplaceAll(h.Text, " ", "_"))
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
	code := string(c)
	if code == "``" {
		code = " `` "
	}
	return "`" + code + "`"
}

func (i Italic) Markdown() string {
	return "*" + string(i) + "*"
}

func (p Pre) Markdown() string {
	return "```\n" + string(p) + "\n```\n"
}

var (
	Cache Spec

	TOC           *api.InteractionResponseData
	Subcomponents = map[string][]discord.SelectOption{}
)

var (
	tocOptions []discord.SelectOption
	GoBack     = discord.SelectOption{
		Label: "Go Back",
		Value: "back",
		Emoji: &discord.ComponentEmoji{Name: "↩️"},
	}
)

func init() {
	var err error
	Cache, err = QuerySpec()
	if err != nil {
		panic(err)
	}
	for i, node := range Cache.Nodes {
		prefix := strconv.Itoa(i+1) + ". "
		tocOptions = append(tocOptions, discord.SelectOption{
			Label: prefix + node.Heading,
			Value: node.Heading,
		})

		Subcomponents[node.Heading] = append(Subcomponents[node.Heading], GoBack)

		for i, sub := range node.Nodes {
			prefix := strconv.Itoa(i+1) + ". "
			Subcomponents[node.Heading] = append(Subcomponents[node.Heading], discord.SelectOption{
				Label: prefix + sub.Heading,
				Value: sub.Heading,
			})
		}
	}
	TOC = &api.InteractionResponseData{
		Flags: api.EphemeralResponse,
		Embeds: &[]discord.Embed{{
			Title: "Spec - Table of Contents",
			Description: `Use the component below to select a subheading.

Search for a full heading to view full heading contents.
**Example**:

/spec query:introduction
/spec query:method sets
/spec query:packages`,
			Color: 0x00ADD8,
		}},
		Components: discord.ComponentsPtr(
			&discord.SelectComponent{
				CustomID:    "spec.toc",
				Placeholder: "View Headings",
				Options:     tocOptions,
			},
		),
	}
}

func NodesSelect(nodes []*Node) *discord.ContainerComponents {
	var options []discord.SelectOption

	options = append(options, GoBack)

	for i, node := range nodes {
		prefix := strconv.Itoa(i+1) + ". "
		options = append(options, discord.SelectOption{
			Label: prefix + node.Heading,
			Value: node.Heading,
		})
	}

	return discord.ComponentsPtr(
		&discord.SelectComponent{
			Placeholder: "Select",
			CustomID:    "spec.toc",
			Options:     options,
		},
	)
}
