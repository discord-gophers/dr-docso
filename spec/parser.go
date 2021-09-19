package spec

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

const (
	page = "https://golang.org/ref/spec"
)

func QuerySpec() (Spec, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(spec))
	if err != nil {
		return Spec{}, fmt.Errorf("could not parse body: %w", err)
	}

	spec := Spec{
		keywords: make(map[string]map[*Node]struct{}),
		Keywords: make(map[string][]*Node),
		Headings: make(map[string]*Node),
	}
	var h2Node, h3Node, h4Node *Node

	add := func(note Note) {
		var node *Node
		switch {
		case h4Node != nil:
			node = h4Node
			h4Node.Content = append(h4Node.Content, note)
		case h3Node != nil:
			node = h3Node
			h3Node.Content = append(h3Node.Content, note)
		case h2Node != nil:
			node = h2Node
			h2Node.Content = append(h2Node.Content, note)
		}
		spec.addKeywords(note, node)
	}

	all := doc.Find("#nav").NextAll()
	all.Each(func(i int, s *goquery.Selection) {
		node := all.Get(i)

		switch node.Data {
		case "h2":
			if h2Node != nil {
				if h3Node != nil {
					if h4Node != nil {
						h3Node.Nodes = append(h3Node.Nodes, h4Node)
					}
					h2Node.Nodes = append(h2Node.Nodes, h3Node)
				}
				spec.Nodes = append(spec.Nodes, h2Node)
			}
			text := s.Text()
			h2Node, h3Node, h4Node = &Node{2, text, nil, nil}, nil, nil
			spec.Headings[text] = h2Node
			add(Heading{2, s.Text()})
		case "h3":
			if h3Node != nil {
				if h4Node != nil {
					h3Node.Nodes = append(h3Node.Nodes, h4Node)
				}
				h2Node.Nodes = append(h2Node.Nodes, h3Node)
			}

			text := s.Text()
			h3Node, h4Node = &Node{Level: 3, Heading: text}, nil
			spec.Headings[text] = h3Node
			add(Heading{3, text})

		case "h4":
			if h4Node != nil {
				h3Node.Nodes = append(h3Node.Nodes, h4Node)
			}

			text := s.Text()
			h4Node = &Node{Level: 4, Heading: text}
			spec.Headings[text] = h4Node
			add(Heading{4, text})

		case "pre":
			add(Pre(strings.TrimSpace(s.Text())))
		case "ul":
			add(List{
				Ordered: false,
				Items:   parseList(node),
			})

		case "ol":
			add(List{
				Ordered: true,
				Items:   parseList(node),
			})

		case "p":
			add(parseText(node))
		}
	})
	if h2Node != nil {
		if h3Node != nil {
			if h4Node != nil {
				h3Node.Nodes = append(h3Node.Nodes, h4Node)
			}
			h2Node.Nodes = append(h2Node.Nodes, h3Node)
		}
		spec.Nodes = append(spec.Nodes, h2Node)
	}

	spec.keywords = nil
	return spec, nil
}

func (s Spec) addKeywords(note Note, node *Node) {
	var text string
	switch v := note.(type) {
	case Heading:
		text = v.Text
	case Link:
		text = v.Text
	case List:
		for _, n := range v.Items {
			s.addKeywords(n, node)
		}
	case Paragraph:
		for _, n := range v {
			s.addKeywords(n, node)
		}
	case Code:
		text = string(v)
	case Italic:
		text = string(v)
	case Pre:
		text = string(v)
	case Text:
		text = string(v)
	}

	for _, f := range strings.Fields(text) {
		key := strings.ToLower(f)
		val := s.keywords[key]
		if val == nil {
			s.keywords[key] = make(map[*Node]struct{})
		}

		if _, ok := s.keywords[key][node]; ok {
			continue
		}

		s.keywords[key][node] = struct{}{}
		s.Keywords[key] = append(s.Keywords[key], node)
	}
}

func parseList(node *html.Node) (items []Paragraph) {
	for li := node.FirstChild; li != nil; li = li.NextSibling {
		if li.Data != "li" {
			continue
		}
		items = append(items, parseText(li))
	}
	return
}

func parseText(node *html.Node) (p Paragraph) {
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		switch n.Type {
		case html.TextNode:
			// split := strings.Split(n.Data, "\n")
			str := strings.ReplaceAll(n.Data, "\n", " ")
			p = append(p, Text(str))

			// var str []string
			// for _, line := range split {
			// 	if line == "" {
			// 		continue
			// 	}

			// 	// Dont trim whitespace from the first element. The first
			// 	// element will be newline if it needs to be trimmed
			// 	// i.e: <p>Text <a>..</a> and more text</p>
			// 	//                       ^
			// 	for strings.HasPrefix(line, " ") && n.PrevSibling == nil {
			// 		line = line[1:]
			// 	}
			// 	str = append(str, line)
			// }
			// p = append(p, Text(strings.Join(str, " ")))

		case html.ElementNode:
			switch n.Data {
			case "a":
				p = append(p, parseA(n))
			case "i":
				p = append(p, Italic(n.FirstChild.Data))
			case "code":
				p = append(p, Code(n.FirstChild.Data))
			}
		}
	}
	return
}

func parseA(a *html.Node) Link {
	var location string
	for _, attr := range a.Attr {
		if attr.Key != "href" {
			continue
		}

		if strings.HasPrefix(attr.Val, "#") {
			attr.Val = page + attr.Val
		}

		if strings.HasPrefix(attr.Val, "/") {
			attr.Val = "https://golang.org" + attr.Val
		}

		location = attr.Val
	}
	text := a.FirstChild.Data
	// <code> inside of an <a>
	if a.FirstChild.Type == html.ElementNode {
		text = a.FirstChild.FirstChild.Data
	}
	return Link{
		Text:     text,
		Location: location,
	}
}

//go:embed spec.html
var spec string
