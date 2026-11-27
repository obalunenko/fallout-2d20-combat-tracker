package mcpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// skillsClient connects a client that can send skills/list and skills/get.
func skillsClient(t *testing.T, env *testEnv) *mcp.ClientSession {
	t.Helper()
	c := mcp.NewClient(&mcp.Implementation{Name: "skills-client", Version: "test"}, nil)
	require.NoError(t, mcp.AddSendingCustomMethod[*skillsListParams, *skillsListResult](c, "skills/list"))
	require.NoError(t, mcp.AddSendingCustomMethod[*skillsGetParams, *skillsGetResult](c, "skills/get"))
	transport := &mcp.StreamableClientTransport{
		Endpoint:   env.url,
		HTTPClient: &http.Client{Transport: bearerTransport{token: string(testSecret), base: http.DefaultTransport}},
	}
	cs, err := c.Connect(t.Context(), transport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestSkillsExtension(t *testing.T) {
	env := newTestEnv(t)
	cs := skillsClient(t, env)

	caps := cs.InitializeResult().Capabilities
	require.NotNil(t, caps.Resources, "resources capability accompanies the extension")
	require.Equal(t, map[string]any{"directoryRead": false}, caps.Extensions[skillsExtensionID])

	list, err := mcp.CallCustomMethod[*skillsListParams, *skillsListResult](t.Context(), cs, "skills/list", &skillsListParams{})
	require.NoError(t, err)
	require.Equal(t, "complete", list.ResultType)
	require.Positive(t, list.TTLMs)
	require.NotEmpty(t, list.CacheScope)
	require.Len(t, list.Skills, 3)

	for _, s := range list.Skills {
		name := s.Frontmatter["name"].(string)
		require.Regexp(t, `^[a-z0-9]+(-[a-z0-9]+)*$`, name)
		require.LessOrEqual(t, len(name), 64)
		require.Equal(t, "skill://"+name+"/SKILL.md", s.URI, "the last path segment equals the name")
		desc := s.Frontmatter["description"].(string)
		require.NotEmpty(t, desc)
		require.LessOrEqual(t, len(desc), 1024)
		require.Equal(t, "test", s.Frontmatter["metadata"].(map[string]any)["version"], "version injected from the build")

		read, err := cs.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: s.URI})
		require.NoError(t, err)
		body := read.Contents[0].Text
		sum := sha256.Sum256([]byte(body))
		require.Len(t, s.Resources, 1)
		require.Equal(t, "sha256:"+hex.EncodeToString(sum[:]), s.Resources[0].Digest, "digest is computed from the served bytes")
		require.Equal(t, len(body), s.Resources[0].Size)
		require.Less(t, strings.Count(body, "\n"), 500)

		got, err := mcp.CallCustomMethod[*skillsGetParams, *skillsGetResult](t.Context(), cs, "skills/get", &skillsGetParams{URI: s.URI})
		require.NoError(t, err)
		require.Equal(t, s.URI, got.Skill.URI)
	}

	_, err = mcp.CallCustomMethod[*skillsGetParams, *skillsGetResult](t.Context(), cs, "skills/get", &skillsGetParams{URI: "skill://nope/SKILL.md"})
	var wire *jsonrpc.Error
	require.True(t, errors.As(err, &wire), "got %v", err)
	require.EqualValues(t, jsonrpc.CodeInvalidParams, wire.Code)
}

func TestSkillsAreOrdinaryResourcesToo(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	uris := map[string]bool{}
	for r, err := range cs.Resources(t.Context(), nil) {
		require.NoError(t, err)
		uris[r.URI] = true
	}
	for _, want := range []string{"skill://start-campaign/SKILL.md", "skill://prepare-encounter/SKILL.md", "skill://run-combat-round/SKILL.md", rulesURI} {
		require.True(t, uris[want], want)
	}
	rules, err := cs.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: rulesURI})
	require.NoError(t, err)
	require.Contains(t, rules.Contents[0].Text, "Torso-only")
}

func TestServerInstructions(t *testing.T) {
	require.Less(t, len(serverInstructions), 1500)
	for _, want := range []string{"capped at 6", "confirm", "makes it the active encounter"} {
		require.Contains(t, serverInstructions, want)
	}
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	require.Equal(t, serverInstructions, cs.InitializeResult().Instructions)
}
