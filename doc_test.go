package main

import "testing"

func TestParseQuery(t *testing.T) {
	cases := []struct {
		name   string
		query  string
		module string
		parts  []string
	}{
		{
			name:   "stdlib basic",
			query:  "strings",
			module: "strings",
			parts:  nil,
		},
		{
			name:   "stdlib type",
			query:  "strings.Split",
			module: "strings",
			parts:  []string{"split"},
		},
		{
			name:   "stdlib method",
			query:  "strings.Builder.Grow",
			module: "strings",
			parts:  []string{"builder", "grow"},
		},
		{
			name:   "stdlib redirect basic",
			query:  "json",
			module: "encoding/json",
			parts:  nil,
		},
		{
			name:   "stdlib redirect type",
			query:  "json.Unmarshal",
			module: "encoding/json",
			parts:  []string{"unmarshal"},
		},
		{
			name:   "stdlib redirect method",
			query:  "json.NewDecoder.Decode",
			module: "encoding/json",
			parts:  []string{"newdecoder", "decode"},
		},
		{
			name:   "custom basic",
			query:  "github.com/golang/go",
			module: "github.com/golang/go",
			parts:  nil,
		},
		{
			name:   "custom type",
			query:  "github.com/bwmarrin/discordgo.Session",
			module: "github.com/bwmarrin/discordgo",
			parts:  []string{"session"},
		},
		{
			name:   "custom method",
			query:  "github.com/bwmarrin/discordgo.Session.AddHandler",
			module: "github.com/bwmarrin/discordgo",
			parts:  []string{"session", "addhandler"},
		},
		{
			name:   "custom method with space",
			query:  "github.com/bwmarrin/discordgo Session AddHandler",
			module: "github.com/bwmarrin/discordgo",
			parts:  []string{"session", "addhandler"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			module, parts := parseQuery(c.query)
			if module != c.module {
				t.Errorf("INVALID MODULE:\nGOT:%s\nEXPECTED:%s", module, c.module)
			}
			if len(parts) != len(c.parts) {
				t.Errorf("INVALID PARTS:\nGOT:%v\nEXPECTED:%v", parts, c.parts)
			}
			for i, part := range parts {
				if part != c.parts[i] {
					t.Errorf("INVALID PARTS(%d):\nGOT:%v\nEXPECTED:%v", i, part, c.parts[i])
				}
			}
		})
	}
}
