package main

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/hhhapz/doc"
)

var (
	searchErr      = "Could not find package with the name of `%s`."
	notFound       = "Could not find type or function `%s` in package `%s`"
	methodNotFound = "Could not find method `%s` for type `%s` in package `%s`"
	notOwner       = "Only the message sender can do this."
	cannotExpand   = "You cannot expand this embed."
)

type interactionData struct {
	id      string
	created time.Time
	token   string
	userID  discord.UserID
	query   string
	full    bool
}

var (
	interactionMap = map[string]*interactionData{}
	mu             sync.Mutex
)

func (b *botState) gcInteractionData() {
	mapTicker := time.NewTicker(time.Minute * 5)
	cacheTicker := time.NewTicker(time.Hour * 24)
	for {
		select {

		// gc interaction tokens
		case <-mapTicker.C:
			now := time.Now()
			mu.Lock()
			for _, data := range interactionMap {
				if !now.After(data.created.Add(time.Minute * 5)) {
					continue
				}
				delete(interactionMap, data.id)
				b.state.EditInteractionResponse(b.appID, data.token, api.EditInteractionResponseData{
					Components: &[]discord.Component{},
				})
			}
			mu.Unlock()

		case <-cacheTicker.C:
			b.searcher.WithCache(func(cache map[string]*doc.CachedPackage) {
				for k := range cache {
					delete(cache, k)
				}
			})
		}
	}
}

func (b *botState) handleDocs(e *gateway.InteractionCreateEvent, d *discord.CommandInteractionData) {
	data := api.InteractionResponse{Type: api.DeferredMessageInteractionWithSource}
	if err := b.state.RespondInteraction(e.ID, e.Token, data); err != nil {
		log.Println(fmt.Errorf("could not send interaction callback, %w", err))
		return
	}

	args := map[string]discord.InteractionOption{}
	for _, arg := range d.Options {
		args[arg.Name] = arg
	}

	query := args["query"].String()
	embed := b.onDocs(e, query, false)

	if strings.HasPrefix(embed.Title, "Error") {
		err := b.state.DeleteInteractionResponse(e.AppID, e.Token)
		if err != nil {
			log.Println("failed to delete message:", err)
			return
		}
		_, _ = b.state.CreateInteractionFollowup(e.AppID, e.Token, api.InteractionResponseData{
			Flags:  api.EphemeralResponse,
			Embeds: &[]discord.Embed{embed},
		})
		return
	}

	mu.Lock()
	interactionMap[e.ID.String()] = &interactionData{
		id:      e.ID.String(),
		created: time.Now(),
		token:   e.Token,
		userID:  e.User.ID,
		query:   query,
	}
	mu.Unlock()

	if _, err := b.state.EditInteractionResponse(e.AppID, e.Token, api.EditInteractionResponseData{
		Embeds: &[]discord.Embed{embed},
		Components: &[]discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{
					&discord.SelectComponent{
						CustomID:    e.ID.String(),
						Options:     selectOptions(false),
						Placeholder: "Actions",
					},
				},
			},
		},
	}); err != nil {
		log.Println(fmt.Errorf("could not send interaction callback, %w", err))
		return
	}
}

func (b *botState) onDocsComponent(e *gateway.InteractionCreateEvent, data *interactionData) {
	var embed discord.Embed
	var components *[]discord.Component

	// if e.Member is nil, all operations should be allowed
	hasRole := e.Member == nil
	if !hasRole {
		for _, role := range e.Member.RoleIDs {
			if _, ok := b.cfg.Permissions.Docs[role]; ok {
				hasRole = true
				break
			}
		}
	}

	hasPerm := func() bool {
		if hasRole {
			return true
		}

		perms, err := b.state.Permissions(e.ChannelID, e.User.ID)
		if err != nil {
			return false
		}
		if !perms.Has(discord.PermissionAdministrator) {
			return false
		}
		return true
	}

	action := e.Data.(*discord.ComponentInteractionData).Values[0]
	switch action {
	case "minimize":
		embed, data.full = b.onDocs(e, data.query, false), false
		components = &[]discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{
					&discord.SelectComponent{
						CustomID:    data.id,
						Options:     selectOptions(false),
						Placeholder: "Actions",
					},
				},
			},
		}

	// Admin or privileged only.
	// (Only check admin here to reduce total API calls).
	// If not privileged, send ephemeral instead.
	case "expand":
		embed, data.full = b.onDocs(e, data.query, true), true
		components = &[]discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{
					&discord.SelectComponent{
						CustomID:    data.id,
						Options:     selectOptions(true),
						Placeholder: "Actions",
					},
				},
			},
		}

		if !hasPerm() {
			_ = b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
				Type: api.MessageInteractionWithSource,
				Data: &api.InteractionResponseData{
					Flags:  api.EphemeralResponse,
					Embeds: &[]discord.Embed{embed},
				},
			})
			break
		}

	case "hide":
		components = &[]discord.Component{}
		embed = b.onDocs(e, data.query, data.full)
		embed.Description = ""
		embed.Footer = nil
		mu.Lock()
		delete(interactionMap, data.id)
		mu.Unlock()
	}

	if e.GuildID != discord.NullGuildID {
		// Check admin last.
		if e.User.ID != data.userID && !hasPerm() {
			embed = failEmbed("Error", notOwner)
		}
	}

	var resp api.InteractionResponse
	if strings.HasPrefix(embed.Title, "Error") {
		resp = api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		}
	} else {
		resp = api.InteractionResponse{
			Type: api.UpdateMessage,
			Data: &api.InteractionResponseData{
				Embeds:     &[]discord.Embed{embed},
				Components: components,
			},
		}
	}
	b.state.RespondInteraction(e.ID, e.Token, resp)
}

func (b *botState) onDocs(e *gateway.InteractionCreateEvent, query string, full bool) discord.Embed {
	module, parts := parseQuery(query)
	pkg, err := b.searcher.Search(context.Background(), module)
	if err != nil {
		log.Printf("Package request by %s failed: %v", e.User.Tag(), err)
		return failEmbed("Error", fmt.Sprintf(searchErr, module))
	}

	switch len(parts) {
	case 0:
		return b.fullPackage(pkg, full)
	case 1:
		if typ, ok := pkg.Types[parts[0]]; ok {
			return b.typ(pkg, typ, full)
		}

		if fn, ok := pkg.Functions[parts[0]]; ok {
			return b.fn(pkg, fn, full)
		}

		return failEmbed("Error: Not Found", fmt.Sprintf(notFound, parts[0], module))
	default:
		typ, ok := pkg.Types[parts[0]]
		if !ok {
			return failEmbed("Error: Not Found", fmt.Sprintf(notFound, parts[0], module))
		}

		method, ok := typ.Methods[parts[1]]
		if !ok {
			return failEmbed("Error: Not Found", fmt.Sprintf(methodNotFound, parts[1], parts[0], module))
		}

		return b.method(pkg, method, full)
	}
}

const (
	docLimit = 2800
	defLimit = 1000

	accentColor = 0x007D9C
)

func (b *botState) fullPackage(pkg doc.Package, full bool) discord.Embed {
	return discord.Embed{
		Title: "Package " + pkg.URL,
		URL:   "https://pkg.go.dev/" + pkg.URL,
		Description: fmt.Sprintf("**Types:** %d\n**Functions:** %d\n\n%s",
			len(pkg.Types), len(pkg.Functions), format(pkg.Overview, 32, full)),
		Color: accentColor,
		Footer: &discord.EmbedFooter{
			Text: "https://pkg.go.dev/" + pkg.URL,
		},
	}
}

func (b *botState) typ(pkg doc.Package, typ doc.Type, full bool) discord.Embed {
	def := typdef(typ.Signature, full)
	return discord.Embed{
		Title:       fmt.Sprintf("%s: %s", pkg.URL, typ.Name),
		URL:         fmt.Sprintf("https://pkg.go.dev/%s#%s", pkg.URL, typ.Name),
		Description: fmt.Sprintf("```go\n%s\n```\n%s", def, format(typ.Comment, len(def), full)),
		Color:       accentColor,
		Footer: &discord.EmbedFooter{
			Text: "https://pkg.go.dev/" + pkg.URL,
		},
	}
}

func (b *botState) fn(pkg doc.Package, fn doc.Function, full bool) discord.Embed {
	def := typdef(fn.Signature, full)
	return discord.Embed{
		Title:       fmt.Sprintf("%s: %s", pkg.URL, fn.Name),
		URL:         fmt.Sprintf("https://pkg.go.dev/%s#%s", pkg.URL, fn.Name),
		Description: fmt.Sprintf("```go\n%s\n```\n%s", def, format(fn.Comment, len(def), full)),
		Color:       accentColor,
		Footer: &discord.EmbedFooter{
			Text: "https://pkg.go.dev/" + pkg.URL,
		},
	}
}

func (b *botState) method(pkg doc.Package, method doc.Method, full bool) discord.Embed {
	def := typdef(method.Signature, full)
	return discord.Embed{
		Title:       fmt.Sprintf("%s: %s.%s", pkg.URL, method.For, method.Name),
		URL:         fmt.Sprintf("https://pkg.go.dev/%s#%s.%s", pkg.URL, method.For, method.Name),
		Description: fmt.Sprintf("```go\n%s\n```\n%s", def, format(method.Comment, len(def), full)),
		Color:       accentColor,
		Footer: &discord.EmbedFooter{
			Text: "https://pkg.go.dev/" + pkg.URL,
		},
	}
}

func selectOptions(full bool) []discord.SelectComponentOption {
	expand := discord.SelectComponentOption{
		Label:       "Expand",
		Value:       "expand",
		Description: "Show more documentation.",
		Emoji:       &discord.ButtonEmoji{Name: "⬇️"},
	}
	if full {
		expand = discord.SelectComponentOption{
			Label:       "Minimize",
			Value:       "minimize",
			Description: "Show less documentation.",
			Emoji:       &discord.ButtonEmoji{Name: "⬆️"},
		}
	}

	return []discord.SelectComponentOption{
		expand,
		{
			Label:       "Hide",
			Value:       "hide",
			Description: "Hide the message.",
			Emoji:       &discord.ButtonEmoji{Name: "❌"},
		},
	}
}

func failEmbed(title, description string) discord.Embed {
	return discord.Embed{
		Title:       title,
		Description: description,
		Color:       0xEE0000,
	}
}

func parseQuery(module string) (string, []string) {
	module = strings.ReplaceAll(module, " ", ".")
	dir, base := path.Split(strings.ToLower(module))
	split := strings.Split(base, ".")
	full := dir + split[0]
	if strings.HasPrefix(full, "x/") {
		full = "golang.org/" + full
	}

	if complete, ok := stdlibPackages[full]; ok {
		full = complete
	}
	return full, split[1:]
}

func typdef(def string, full bool) string {
	split := strings.Split(def, "\n")
	if !full {
		return split[0]
	}

	b := strings.Builder{}
	b.Grow(len(def))

	for _, line := range strings.Split(def, "\n") {
		b.WriteRune('\n')

		if len(line)+b.Len() > defLimit {
			b.WriteString("// full signature omitted")
			break
		}
		b.WriteString(line)
	}
	return b.String()
}

func format(c doc.Comment, initial int, full bool) string {
	if len(c) == 0 {
		return "*No documentation found*"
	}

	if !full {
		if md := c.Markdown(); len(md) < 500 {
			return md
		}

		md := c[0].Markdown()
		if len(md) > 500 {
			md = md[:400] + "...\n\n*More documentation omitted*"
		}
		if len(c) == 1 {
			return md
		}
		return fmt.Sprintf("%s\n\n*More documentation omitted*", md)
	}

	var parts doc.Comment
	length := initial
	for _, note := range c {
		l := len(note.Text())
		if l+length > docLimit {
			parts = append(parts, doc.Paragraph("*More documentation omitted...*"))
			break
		}
		length += l
		parts = append(parts, note)
	}
	return parts.Markdown()
}

var stdlibPackages = map[string]string{
	"tar": "archive/tar",
	"zip": "archive/zip",

	"bzip2": "compress/bzip2",
	"flate": "compress/flate",
	"gzip":  "compress/gzip",
	"lzw":   "compress/lzw",
	"zlib":  "compress/zlib",

	"heap": "container/heap",
	"list": "container/list",
	"ring": "container/ring",

	"aes":      "crypto/aes",
	"cipher":   "crypto/cipher",
	"des":      "crypto/des",
	"dsa":      "crypto/dsa",
	"ecdsa":    "crypto/ecdsa",
	"ed25519":  "crypto/ed25519",
	"elliptic": "crypto/elliptic",
	"hmac":     "crypto/hmac",
	"md5":      "crypto/md5",
	"rc4":      "crypto/rc4",
	"rsa":      "crypto/rsa",
	"sha1":     "crypto/sha1",
	"sha256":   "crypto/sha256",
	"sha512":   "crypto/sha512",
	"subtle":   "crypto/subtle",
	"tls":      "crypto/tls",
	"x509":     "crypto/x509",
	"pkix":     "crypto/x509/pkix",

	"dwarf":    "debug/dwarf",
	"elf":      "debug/elf",
	"gosym":    "debug/gosym",
	"macho":    "debug/macho",
	"pe":       "debug/pe",
	"plan9obj": "debug/plan9obj",

	"ascii85": "encoding/ascii85",
	"asn1":    "encoding/asn1",
	"base32":  "encoding/base32",
	"base64":  "encoding/base64",
	"binary":  "encoding/binary",
	"csv":     "encoding/csv",
	"gob":     "encoding/gob",
	"hex":     "encoding/hex",
	"json":    "encoding/json",
	"pem":     "encoding/pem",
	"xml":     "encoding/xml",

	"ast":           "go/ast",
	"build":         "go/build",
	"constraint":    "go/build/constraint",
	"constant":      "go/constant",
	"docformat":     "go/docformat",
	"importer":      "go/importer",
	"parserprinter": "go/parserprinter",
	"scanner":       "go/scanner",
	"token":         "go/token",
	"types":         "go/types",

	"adler32": "hash/adler32",
	"crc32":   "hash/crc32",
	"crc64":   "hash/crc64",
	"fnv":     "hash/fnv",
	"maphash": "hash/maphash",

	"color":   "image/color",
	"draw":    "image/draw",
	"gif":     "image/gif",
	"jpeg":    "image/jpeg",
	"parsing": "image/parsing",

	"suffixarray": "index/suffixarray",

	"fs":     "io/fs",
	"ioutil": "io/ioutil",

	"big":   "math/big",
	"bits":  "math/bits",
	"cmplx": "math/cmplx",

	"multipart":       "mime/multipart",
	"quotedprintable": "mime/quotedprintable",

	"http":      "net/http",
	"cgi":       "net/http/cgi",
	"cookiejar": "net/http/cookiejar",
	"fcgi":      "net/http/fcgi",
	"httptest":  "net/http/httptest",
	"httptrace": "net/http/httptrace",
	"httputil":  "net/http/httputil",
	"mail":      "net/mail",
	"rpc":       "net/rpc",
	"jsonrpc":   "net/rpc/jsonrpc",
	"smtp":      "net/smtp",
	"textproto": "net/textproto",

	"exec":   "os/exec",
	"signal": "os/signal",
	"user":   "os/user",

	"filepath": "path/filepath",

	"syntax": "regexp/syntax",

	"cgo":     "runtime/cgo",
	"metrics": "runtime/metrics",
	"msan":    "runtime/msan",
	"race":    "runtime/race",
	"trace":   "runtime/trace",

	"js": "syscall/js",

	"fstest": "testing/fstest",
	"iotest": "testing/iotest",
	"quick":  "testing/quick",

	"tabwriter": "text/tabwriter",

	"parse": "text/template/parse",

	"tzdata": "time/tzdata",

	"utf16": "unicode/utf16",
	"utf8":  "unicode/utf8",
}
