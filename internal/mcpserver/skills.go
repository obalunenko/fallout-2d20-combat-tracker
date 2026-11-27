package mcpserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// skillsExtensionID is the Skills extension (SEP-2640). The SDK has no
// released support yet, so this file serves it by hand: skills/list,
// skills/get and the skill files as ordinary resources. Replacing it with the
// SDK package later only touches this file.
const skillsExtensionID = "io.modelcontextprotocol/skills"

const (
	rulesURI     = "fallout://reference/rules"
	markdownMIME = "text/markdown"
	versionLine  = "version: dev"
)

//go:embed skills reference
var embedded embed.FS

var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// skill is one served skill. body is exactly what resources/read returns,
// and digest and size are computed from it.
type skill struct {
	name        string
	uri         string
	description string
	frontmatter map[string]any
	body        []byte
	digest      string
}

// loadSkills reads the embedded skills and injects version into their
// metadata. Digests are computed after injection, so a tracker upgrade
// changes them and clients ask the GM to approve the new skills.
func loadSkills(version string) ([]skill, error) {
	dirs, err := fs.ReadDir(embedded, "skills")
	if err != nil {
		return nil, err
	}
	out := make([]skill, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		raw, err := embedded.ReadFile(path.Join("skills", dir.Name(), "SKILL.md"))
		if err != nil {
			return nil, err
		}
		body := bytes.Replace(raw, []byte(versionLine), []byte("version: "+yamlQuote(version)), 1)
		fm, err := parseFrontmatter(body)
		if err != nil {
			return nil, fmt.Errorf("skill %s: %w", dir.Name(), err)
		}
		name, _ := fm["name"].(string)
		description, _ := fm["description"].(string)
		if name != dir.Name() || !skillNamePattern.MatchString(name) || len(name) > 64 {
			return nil, fmt.Errorf("skill %s: name %q must equal its directory and match %s", dir.Name(), name, skillNamePattern)
		}
		if description == "" || len(description) > 1024 {
			return nil, fmt.Errorf("skill %s: description must be 1 to 1024 characters", name)
		}
		sum := sha256.Sum256(body)
		out = append(out, skill{
			name:        name,
			uri:         "skill://" + name + "/SKILL.md",
			description: description,
			frontmatter: fm,
			body:        body,
			digest:      "sha256:" + hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func yamlQuote(s string) string {
	b, err := yaml.Marshal(s)
	if err != nil {
		return `"dev"`
	}
	return string(bytes.TrimSpace(b))
}

func parseFrontmatter(body []byte) (map[string]any, error) {
	rest, ok := bytes.CutPrefix(body, []byte("---\n"))
	if !ok {
		return nil, fmt.Errorf("missing frontmatter")
	}
	head, _, ok := bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		return nil, fmt.Errorf("unterminated frontmatter")
	}
	fm := map[string]any{}
	if err := yaml.Unmarshal(head, &fm); err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}
	return fm, nil
}

type skillFileRef struct {
	URI    string `json:"uri"`
	Digest string `json:"digest"`
	Size   int    `json:"size"`
}

type skillEntry struct {
	URI         string         `json:"uri"`
	Frontmatter map[string]any `json:"frontmatter"`
	Resources   []skillFileRef `json:"resources"`
}

type skillsListParams struct {
	mcp.ParamsBase
	Cursor string `json:"cursor,omitempty"`
}

type skillsListResult struct {
	mcp.ResultBase
	mcp.Cacheable
	ResultType string       `json:"resultType"`
	Skills     []skillEntry `json:"skills"`
}

type skillsGetParams struct {
	mcp.ParamsBase
	URI string `json:"uri"`
}

type skillsGetResult struct {
	mcp.ResultBase
	ResultType string     `json:"resultType"`
	Skill      skillEntry `json:"skill"`
}

func (s skill) entry() skillEntry {
	return skillEntry{
		URI:         s.uri,
		Frontmatter: s.frontmatter,
		Resources:   []skillFileRef{{URI: s.uri, Digest: s.digest, Size: len(s.body)}},
	}
}

// registerSkills serves the skills over the Skills extension and, for every
// client, as ordinary resources together with the rules reference (FR-019).
// The caller advertises the extension in the server capabilities.
func registerSkills(s *mcp.Server, skills []skill) error {
	for _, sk := range skills {
		s.AddResource(&mcp.Resource{URI: sk.uri, Name: sk.name, Title: "Playbook: " + sk.name, Description: sk.description, MIMEType: markdownMIME},
			staticResource(sk.uri, sk.body))
	}
	rules, err := embedded.ReadFile("reference/rules.md")
	if err != nil {
		return err
	}
	s.AddResource(&mcp.Resource{URI: rulesURI, Name: "rules", Title: "Fallout 2d20 tracker rules", Description: "Difficulty formula and bands, Party AP cap, GM Threat, DR model, immunity, torso-only creatures, confirmations.", MIMEType: markdownMIME},
		staticResource(rulesURI, rules))

	entries := make([]skillEntry, 0, len(skills))
	byURI := make(map[string]skillEntry, len(skills))
	for _, sk := range skills {
		e := sk.entry()
		entries = append(entries, e)
		byURI[sk.uri] = e
	}
	if err := mcp.AddReceivingCustomMethod(s, "skills/list", func(context.Context, *mcp.ServerSession, *skillsListParams) (*skillsListResult, error) {
		return &skillsListResult{Cacheable: mcp.Cacheable{TTLMs: listCacheTTLMs, CacheScope: "public"}, ResultType: "complete", Skills: entries}, nil
	}); err != nil {
		return err
	}
	return mcp.AddReceivingCustomMethod(s, "skills/get", func(_ context.Context, _ *mcp.ServerSession, p *skillsGetParams) (*skillsGetResult, error) {
		e, ok := byURI[p.URI]
		if !ok {
			return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: fmt.Sprintf("unknown skill %q", p.URI)}
		}
		return &skillsGetResult{ResultType: "complete", Skill: e}, nil
	})
}

func staticResource(uri string, body []byte) mcp.ResourceHandler {
	text := string(body)
	return func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: markdownMIME, Text: text}}}, nil
	}
}
