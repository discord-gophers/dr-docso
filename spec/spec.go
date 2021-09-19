package spec

type Spec struct {
	Nodes    []*Node
	keywords map[string]map[*Node]struct{}
	Keywords map[string][]*Node
	Headings map[string]*Node
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
