// Package mcpserver is the Model Context Protocol adapter of the Fallout 2d20
// combat tracker. It serves tools, resources and skills over a loopback-only
// Streamable HTTP endpoint embedded in the desktop process, and provides the
// stdio bridge used by clients that cannot reach a local address.
//
// The package may import internal/app, internal/domain and the MCP SDK
// (github.com/modelcontextprotocol/go-sdk/...). It must not import
// internal/store/... or internal/ui/...: every business rule lives in
// internal/app and internal/domain, and the desktop UI is reached only through
// the ports declared in internal/app.
package mcpserver
