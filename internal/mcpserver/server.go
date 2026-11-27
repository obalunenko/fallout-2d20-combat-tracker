package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

// listCacheTTLMs is how long clients may cache list results. Tool, resource
// and skill lists are fixed for the lifetime of the process.
const listCacheTTLMs = 5 * 60 * 1000

// serverInstructions orients an assistant that has no playbook loaded.
// Keep it under 1,500 characters (US5).
const serverInstructions = `Fallout 2d20 combat tracker for a game master (GM). Tools read and change the GM's campaigns, encounters and live combat in the running desktop tracker.
Rules: Party AP is shared and capped at 6; GM Threat has no cap. DR is per body location for physical, energy and radiation damage; poison DR is global; immune types take no damage; torso-only creatures are always hit on the torso.
Destructive tools (delete or restart an encounter, remove combatants, remove a player, mark a character unavailable) ask the GM to confirm; never assume consent.
Saving an encounter makes it the active encounter.
Entities are referenced by id: list first, then use the returned id.
Guides: skill://start-campaign/SKILL.md, skill://prepare-encounter/SKILL.md, skill://run-combat-round/SKILL.md; rules: fallout://reference/rules.`

// deps are what tool handlers need.
type deps struct {
	svc     *app.Service
	confirm *confirmer
	logf    func(string, ...any)
}

// newServer builds the MCP server with every tool, resource and skill.
func newServer(d *deps, version string) (*mcp.Server, error) {
	skills, err := loadSkills(version)
	if err != nil {
		return nil, err
	}
	caps := &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}, Resources: &mcp.ResourceCapabilities{}}
	caps.AddExtension(skillsExtensionID, map[string]any{"directoryRead": false})
	s := mcp.NewServer(&mcp.Implementation{Name: "fallout-tracker", Title: "Fallout 2d20 Combat Tracker", Version: version}, &mcp.ServerOptions{
		Instructions: serverInstructions,
		Capabilities: caps,
		SetCacheable: func(_ context.Context, _ mcp.Request, c *mcp.Cacheable) { c.TTLMs = listCacheTTLMs },
	})
	registerReadTools(s, d)
	registerCampaignTools(s, d)
	registerEncounterTools(s, d)
	registerCombatTools(s, d)
	if err := registerSkills(s, skills); err != nil {
		return nil, err
	}
	return s, nil
}

// newHandler returns the stateless Streamable HTTP handler for s.
func newHandler(s *mcp.Server) http.Handler {
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
	})
	// The SDK keeps context values but drops cancellation for clients older
	// than 2026-07-28. Keeping the request context reachable lets a desktop
	// confirmation end when such a client drops the call.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), httpRequestKey{}, r.Context())))
	})
}

type httpRequestKey struct{}

// requestContext returns the context of the HTTP request that carries ctx's
// call, or ctx when there is none.
func requestContext(ctx context.Context) context.Context {
	if rc, ok := ctx.Value(httpRequestKey{}).(context.Context); ok {
		return rc
	}
	return ctx
}

// effect is what a tool does to stored data (FR-016).
type effect int

const (
	effectRead effect = iota
	effectModify
	effectDestroy
)

type toolDef struct {
	Name        string
	Title       string
	Description string
	Effect      effect
	// Idempotent marks a modifying tool whose repeated call has no further
	// effect. Read tools are always idempotent; destructive ones never are.
	Idempotent bool
}

func (t toolDef) annotations() *mcp.ToolAnnotations {
	no := false
	a := &mcp.ToolAnnotations{Title: t.Title, OpenWorldHint: &no}
	switch t.Effect {
	case effectRead:
		a.ReadOnlyHint, a.IdempotentHint, a.DestructiveHint = true, true, &no
	case effectModify:
		a.IdempotentHint, a.DestructiveHint = t.Idempotent, &no
	case effectDestroy:
		yes := true
		a.DestructiveHint = &yes
	}
	return a
}

// summarizer is implemented by tool outputs that have a one-line summary for
// people (FR-010).
type summarizer interface{ Summary() string }

// toolHandler handles one tool call. A non-nil result is returned as-is; it
// carries input requests (MRTR confirmation). Otherwise out is the
// structured output.
type toolHandler[In, Out any] func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error)

// addTool registers a tool. It validates arguments against the input schema,
// marks the context with the assistant source, and turns errors into the
// tool error contract.
func addTool[In, Out any](s *mcp.Server, d *deps, def toolDef, h toolHandler[In, Out]) {
	// Arguments are checked against the inferred schema. The advertised one
	// adds bounds the domain enforces itself, with messages that name the
	// field and its range (FR-017).
	resolved, err := mustSchema[In]().Resolve(nil)
	if err != nil {
		panic(fmt.Sprintf("tool %s: resolve input schema: %v", def.Name, err))
	}
	tool := &mcp.Tool{
		Name:         def.Name,
		Title:        def.Title,
		Description:  def.Description,
		InputSchema:  constrained[In](true),
		OutputSchema: constrained[Out](false),
		Annotations:  def.annotations(),
	}
	s.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx = app.WithSource(ctx, domain.SourceAssistant)
		args, terr := decodeArgs[In](req.Params.Arguments, resolved)
		if terr != nil {
			if def.Effect != effectRead {
				// A refused change is still an assistant operation (FR-024).
				d.record(ctx, activity{Operation: def.Name}, terr)
			}
			return errorResult(terr), nil
		}
		res, out, err := h(ctx, req, args)
		if err != nil {
			return errorResult(asToolError(err, d.logf)), nil
		}
		if res != nil {
			return res, nil
		}
		return successResult(out)
	})
}

func decodeArgs[In any](raw json.RawMessage, resolved *jsonschema.Resolved) (In, *toolError) {
	var in In
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage("{}")
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return in, invalidInput("", "arguments are not valid JSON")
	}
	if err := resolved.Validate(generic); err != nil {
		return in, invalidInput("", fmt.Sprintf("invalid arguments: %v", err))
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return in, invalidInput("", fmt.Sprintf("invalid arguments: %v", err))
	}
	return in, nil
}

func successResult(out any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal tool output: %w", err)
	}
	text := string(b)
	if s, ok := out.(summarizer); ok {
		text = s.Summary() + "\n" + text
	}
	return &mcp.CallToolResult{
		StructuredContent: json.RawMessage(b),
		Content:           []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil
}

// schemaTypes gives enum schemas to the domain types used in tool inputs.
var schemaTypes = map[reflect.Type]*jsonschema.Schema{
	reflect.TypeFor[domain.DamageType](): {
		Type: "string", Description: "damage type",
		Enum: []any{string(domain.DamagePhysical), string(domain.DamageEnergy), string(domain.DamageRadiation), string(domain.DamagePoison)},
	},
	reflect.TypeFor[domain.BodyLocation](): {
		Type: "string", Description: "body location hit",
		Enum: []any{string(domain.BodyHead), string(domain.BodyTorso), string(domain.BodyLeftArm), string(domain.BodyRightArm), string(domain.BodyLeftLeg), string(domain.BodyRightLeg)},
	},
}

// inputMinimums are the domain lower bounds of integer input fields, by JSON
// name (contracts/mcp-tools.md "Shared DTOs").
var inputMinimums = map[string]float64{
	"level": 1, "max_hp": 1, "quantity": 1, "limit": 1,
	"strength": 1, "perception": 1, "endurance": 1, "charisma": 1, "intelligence": 1, "agility": 1, "luck": 1,
	"xp": 0, "hp": 0, "defense": 0, "amount": 0, "poison": 0,
	"head": 0, "torso": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0,
}

// stringEnums are the closed string fields, by JSON name.
var stringEnums = map[string][]any{
	"side":   {string(domain.SideParty), string(domain.SideNPC)},
	"source": {string(domain.SourceDesktop), string(domain.SourceAssistant)},
}

// constrained is T's schema with what struct inference cannot express: enums
// everywhere and, in input schemas, integer minimums and non-empty
// identifiers (contracts/mcp-tools.md "Inputs").
func constrained[T any](input bool) *jsonschema.Schema {
	s := mustSchema[T]()
	constrain(s, input)
	return s
}

func constrain(s *jsonschema.Schema, input bool) {
	for name, p := range s.Properties {
		switch {
		case input && hasType(p, "integer"):
			if lowest, ok := inputMinimums[name]; ok {
				p.Minimum = new(lowest)
			}
			if name == "quantity" {
				p.Maximum = new(float64(app.MaxDraftMonsterQuantity))
			}
		case hasType(p, "string"):
			if values, ok := stringEnums[name]; ok {
				p.Enum = values
			}
			if input && strings.HasSuffix(name, "_id") {
				p.MinLength = new(1)
			}
		case input && hasType(p, "array") && strings.HasSuffix(name, "_ids") && p.Items != nil:
			p.Items.MinLength = new(1)
		}
		constrain(p, input)
	}
	if s.Items != nil {
		constrain(s.Items, input)
	}
}

func hasType(s *jsonschema.Schema, t string) bool {
	return s.Type == t || slices.Contains(s.Types, t)
}

func mustSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](&jsonschema.ForOptions{TypeSchemas: schemaTypes})
	if err != nil {
		panic(fmt.Sprintf("schema for %T: %v", *new(T), err))
	}
	return s
}

// noInput is the input of tools that take no arguments.
type noInput struct{}
