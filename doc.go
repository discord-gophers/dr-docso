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
	searchErr      = "Could not find package with the name of `%s`.\n\nTry `/docs query:help` for examples."
	notFound       = "Could not find type or function `%s` in package `%s`.\n\nTry `/docs query:help` for examples."
	methodNotFound = "Could not find method `%s` for type `%s` in package `%s`.\n\nTRry `/docs query:help` f`or examples."
	notOwner       = "Only the message sender can do this."
	cannotExpand   = "You cannot expand this embed."
)

type interactionData struct {
	id        string
	created   time.Time
	token     string
	userID    discord.UserID
	messageID discord.MessageID
	query     string
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
			for _, data := range interactionMap {
				if !now.After(data.created.Add(time.Minute * 5)) {
					continue
				}

				mu.Lock()
				delete(interactionMap, data.id)
				mu.Unlock()

				if data.token == "" {
					continue
				}

				b.state.EditInteractionResponse(b.appID, data.token, api.EditInteractionResponseData{
					Components: &[]discord.Component{},
				})
			}

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
		log.Println(fmt.Errorf("could not send interaction callback, %v", err))
		return
	}

	// only arg and required, always present
	query := d.Options[0].String()

	log.Printf("%s used docs(%q)", e.User.Tag(), query)

	var embed discord.Embed
	var internal, more bool
	switch query {
	case "?", "help", "usage":
		embed, internal = helpEmbed(), true
	case "alias", "aliases":
		embed, internal = aliasList(b.cfg.Aliases), true
	default:
		embed, more = b.docs(*e.User, query, false)
	}

	if internal || strings.HasPrefix(embed.Title, "Error") {
		err := b.state.DeleteInteractionResponse(e.AppID, e.Token)
		if err != nil {
			log.Printf("failed to delete message: %v", err)
			return
		}

		// Discord's API means this will always error out, but its still valid.
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

	// If more is true, there is more content that was omitted in the embed.
	// If more is false, there is no more content, and the expand option
	// becomes redundant.
	var component discord.Component = selectComponent(e.ID.String(), false)
	if !more {
		component = buttonComponent(e.ID.String())
	}

	if _, err := b.state.EditInteractionResponse(e.AppID, e.Token, api.EditInteractionResponseData{
		Embeds: &[]discord.Embed{embed},
		Components: &[]discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{component},
			},
		},
	}); err != nil {
		log.Printf("could not send interaction callback, %v", err)
		return
	}
}

func (b *botState) handleDocsText(m *gateway.MessageCreateEvent, query string) {
	log.Printf("%s used docs(%q) text version", m.Author.Tag(), query)

	var embed discord.Embed
	var internal, more bool
	switch query {
	case "?", "help", "usage":
		embed, internal = helpEmbed(), true
	case "alias", "aliases":
		embed, internal = aliasList(b.cfg.Aliases), true
	default:
		embed, more = b.docs(m.Author, query, false)
	}

	if internal {
		b.state.SendEmbedReply(m.ChannelID, m.ID, embed)
		return
	}

	if strings.HasPrefix(embed.Title, "Error") {
		b.state.React(m.ChannelID, m.ID, "😕")
		return
	}

	data, ok := interactionMap[m.ID.String()]
	if ok {
		_, _ = b.state.EditEmbeds(m.ChannelID, data.messageID, embed)
		_ = b.state.DeleteAllReactions(m.ChannelID, data.messageID)
		return
	}

	mu.Lock()
	interactionMap[m.ID.String()] = &interactionData{
		id:      m.ID.String(),
		created: time.Now(),
		userID:  m.Author.ID,
		query:   query,
	}
	mu.Unlock()

	var component discord.Component = selectComponent(m.ID.String(), false)
	if !more {
		component = buttonComponent(m.ID.String())
	}

	msg, err := b.state.SendMessageComplex(m.ChannelID, api.SendMessageData{
		Components: []discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{component},
			},
		},
		Embeds: []discord.Embed{embed},
	})
	if err != nil {
		delete(interactionMap, m.ID.String())
		return
	}
	mu.Lock()
	interactionMap[m.ID.String()].messageID = msg.ID
	mu.Unlock()
}

func (b *botState) handleDocsComponent(e *gateway.InteractionCreateEvent, data *interactionData) {
	var embed discord.Embed
	var components *[]discord.Component

	// if e.Member is nil, all operations should be allowed
	hasRole := e.Member == nil
	if !hasRole {
		for _, role := range e.Member.RoleIDs {
			if _, ok := b.cfg.Permissions.Docs[discord.Snowflake(role)]; ok {
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

	cid := e.Data.(*discord.ComponentInteractionData)

	action := "hide"
	if len(cid.Values) != 0 {
		action = cid.Values[0]
	}

	log.Printf("%s used docs component(%q)", e.User.Tag(), action)

	switch action {
	case "minimize":
		embed, _ = b.docs(*e.User, data.query, false)
		components = &[]discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{selectComponent(data.id, false)},
			},
		}

	// Admin or privileged only.
	// (Only check admin here to reduce total API calls).
	// If not privileged, send ephemeral instead.
	case "expand.all":
		embed, _ = b.docs(*e.User, data.query, true)
		components = &[]discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{selectComponent(data.id, true)},
			},
		}

	case "expand":
		embed, _ = b.docs(*e.User, data.query, true)
		components = &[]discord.Component{
			&discord.ActionRowComponent{
				Components: []discord.Component{selectComponent(data.id, true)},
			},
		}

		_ = b.state.RespondInteraction(e.ID, e.Token, api.InteractionResponse{
			Type: api.MessageInteractionWithSource,
			Data: &api.InteractionResponseData{
				Flags:  api.EphemeralResponse,
				Embeds: &[]discord.Embed{embed},
			},
		})
		return

	case "hide":
		components = &[]discord.Component{}
		embed, _ = b.docs(*e.User, data.query, false)
		embed.Description = ""

		if hasPerm() {
			mu.Lock()
			delete(interactionMap, data.id)
			mu.Unlock()
		}
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

func (b *botState) docs(user discord.User, query string, full bool) (discord.Embed, bool) {
	module, parts := parseQuery(query)
	split := strings.Split(module, "/")
	if full, ok := b.cfg.Aliases[split[0]]; ok {
		split[0] = full
	}

	pkg, err := b.searcher.Search(context.Background(), strings.Join(split, "/"))
	if err != nil {
		log.Printf("Package request by %s(%q) failed: %v", user.Tag(), query, err)
		return failEmbed("Error", fmt.Sprintf(searchErr, module)), false
	}

	switch len(parts) {
	case 0:
		return pkgEmbed(pkg, full)

	case 1:
		if typ, ok := pkg.Types[parts[0]]; ok {
			return typEmbed(pkg, typ, full)
		}
		if fn, ok := pkg.Functions[parts[0]]; ok {
			return fnEmbed(pkg, fn, full)
		}
		return failEmbed("Error: Not Found", fmt.Sprintf(notFound, parts[0], module)), false

	default:
		typ, ok := pkg.Types[parts[0]]
		if !ok {
			return failEmbed("Error: Not Found", fmt.Sprintf(notFound, parts[0], module)), false
		}

		method, ok := typ.Methods[parts[1]]
		if !ok {
			return failEmbed("Error: Not Found", fmt.Sprintf(notFound, parts[1], module)), false
		}

		return methodEmbed(pkg, method, full)
	}
}

func selectComponent(id string, full bool) *discord.SelectComponent {
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

	sel := &discord.SelectComponent{
		CustomID:    id,
		Placeholder: "Actions",
		Options: []discord.SelectComponentOption{
			expand,
			{
				Label:       "Hide",
				Value:       "hide",
				Description: "Hide the message.",
				Emoji:       &discord.ButtonEmoji{Name: "❌"},
			},
		},
	}
	if !full {
		sel.Options = append(sel.Options, discord.SelectComponentOption{
			Label:       "Expand (For everyone)",
			Value:       "expand.all",
			Description: "Show more documentation. (Requires permissions)",
			Emoji:       &discord.ButtonEmoji{Name: "🌏"},
		})
	}

	return sel
}

func buttonComponent(id string) *discord.ButtonComponent {
	return &discord.ButtonComponent{
		CustomID: id,
		Label:    "Hide",
		Emoji:    &discord.ButtonEmoji{Name: "🇽"},
		Style:    discord.SecondaryButton,
	}
}

func parseQuery(module string) (string, []string) {
	var split []string
	var first string

	if strings.Contains(module, "@") {
		split = strings.Split(module, " ")
		first = split[0]
	} else {
		module = strings.ReplaceAll(module, " ", ".")
		dir, base := path.Split(strings.ToLower(module))
		split = strings.Split(base, ".")
		first = dir + split[0]
	}

	if strings.HasPrefix(first, "x/") {
		first = "golang.org/" + first
	}
	if complete, ok := stdlibPackages[first]; ok {
		first = complete
	}

	return first, split[1:]
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

	"sql": "database/sql",

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
