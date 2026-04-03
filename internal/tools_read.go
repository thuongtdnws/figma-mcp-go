package internal

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerReadTools(s *server.MCPServer, node *Node) {
	// ── Document & Selection ─────────────────────────────────────────────

	s.AddTool(mcp.NewTool("get_document",
		mcp.WithDescription("Get the current Figma page document tree"),
	), makeHandler(node, "get_document", nil, nil))

	s.AddTool(mcp.NewTool("get_pages",
		mcp.WithDescription("List all pages in the document with their IDs and names. Lightweight alternative to get_document."),
	), makeHandler(node, "get_pages", nil, nil))

	s.AddTool(mcp.NewTool("get_metadata",
		mcp.WithDescription("Get metadata about the current Figma document: file name, pages, current page"),
	), makeHandler(node, "get_metadata", nil, nil))

	s.AddTool(mcp.NewTool("get_selection",
		mcp.WithDescription("Get the currently selected nodes in Figma"),
	), makeHandler(node, "get_selection", nil, nil))

	s.AddTool(mcp.NewTool("get_node",
		mcp.WithDescription("Get a specific Figma node by ID. Must use colon format e.g. '4029:12345', never hyphens."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Node ID in colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		resp, err := node.Send(ctx, "get_node", []string{nodeID}, nil)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_nodes_info",
		mcp.WithDescription("Get detailed information about multiple Figma nodes by ID in a single call."),
		mcp.WithArray("nodeIds",
			mcp.Required(),
			mcp.Description("List of node IDs in colon format e.g. ['4029:12345', '4029:67890']"),
			mcp.WithStringItems(),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, _ := req.GetArguments()["nodeIds"].([]interface{})
		nodeIDs := toStringSlice(raw)
		resp, err := node.Send(ctx, "get_nodes_info", nodeIDs, nil)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_design_context",
		mcp.WithDescription("Get a depth-limited tree of the current selection or page. More token-efficient than get_document for large files."),
		mcp.WithNumber("depth",
			mcp.Description("How many levels deep to traverse (default 2)"),
		),
		mcp.WithString("detail",
			mcp.Description("Property verbosity: minimal (id/name/type/bounds only), compact (+fills/strokes/opacity), full (everything, default)"),
		),
		mcp.WithBoolean("dedupe_components",
			mcp.Description("When true, INSTANCE nodes are serialized compactly (mainComponentId + componentProperties overrides only) and unique component definitions are collected once in a top-level componentDefs map. Highly token-efficient for screens with many repeated component instances."),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		if d, ok := req.GetArguments()["depth"].(float64); ok && d > 0 {
			params["depth"] = d
		}
		if det, ok := req.GetArguments()["detail"].(string); ok && det != "" {
			params["detail"] = det
		}
		if dd, ok := req.GetArguments()["dedupe_components"].(bool); ok && dd {
			params["dedupeComponents"] = true
		}
		resp, err := node.Send(ctx, "get_design_context", nil, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("search_nodes",
		mcp.WithDescription("Search for nodes by name substring and/or type within a subtree. Avoids dumping the entire document tree."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Name substring to match (case-insensitive)"),
		),
		mcp.WithString("nodeId",
			mcp.Description("Scope search to this subtree (default: current page), colon format e.g. '4029:12345'"),
		),
		mcp.WithArray("types",
			mcp.Description("Filter by Figma node type e.g. ['TEXT', 'FRAME', 'COMPONENT']"),
			mcp.WithStringItems(),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results to return (default: 50)"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{
			"query": req.GetArguments()["query"],
		}
		if id, ok := req.GetArguments()["nodeId"].(string); ok && id != "" {
			params["nodeId"] = id
		}
		if raw, ok := req.GetArguments()["types"].([]interface{}); ok && len(raw) > 0 {
			params["types"] = raw
		}
		if limit, ok := req.GetArguments()["limit"].(float64); ok && limit > 0 {
			params["limit"] = limit
		}
		resp, err := node.Send(ctx, "search_nodes", nil, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("scan_text_nodes",
		mcp.WithDescription("Scan all TEXT nodes in a subtree. Useful for extracting all copy from a component or frame."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Root node ID to scan from, colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		resp, err := node.Send(ctx, "scan_text_nodes", nil, map[string]interface{}{"nodeId": nodeID})
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("scan_nodes_by_types",
		mcp.WithDescription("Find all nodes matching specific types (e.g. FRAME, COMPONENT, INSTANCE) within a subtree."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Root node ID to scan from, colon format e.g. '4029:12345'"),
		),
		mcp.WithArray("types",
			mcp.Required(),
			mcp.Description("Node types to find e.g. ['FRAME', 'COMPONENT', 'INSTANCE']"),
			mcp.WithStringItems(),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		raw, _ := req.GetArguments()["types"].([]interface{})
		resp, err := node.Send(ctx, "scan_nodes_by_types", nil, map[string]interface{}{
			"nodeId": nodeID,
			"types":  raw,
		})
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_reactions",
		mcp.WithDescription("Get prototype/interaction reactions on a node. Useful for understanding interactive prototypes."),
		mcp.WithString("nodeId",
			mcp.Required(),
			mcp.Description("Node ID in colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nodeID, _ := req.GetArguments()["nodeId"].(string)
		resp, err := node.Send(ctx, "get_reactions", []string{nodeID}, nil)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("get_viewport",
		mcp.WithDescription("Get the current Figma viewport: scroll center, zoom level, and visible bounds."),
	), makeHandler(node, "get_viewport", nil, nil))

	s.AddTool(mcp.NewTool("get_fonts",
		mcp.WithDescription("List all fonts used in the current page, sorted by usage frequency. Useful for understanding typography without scanning all text nodes."),
	), makeHandler(node, "get_fonts", nil, nil))

	// ── Styles & Variables ───────────────────────────────────────────────

	s.AddTool(mcp.NewTool("get_styles",
		mcp.WithDescription("Get all local styles in the document: paint, text, effect, and grid styles"),
	), makeHandler(node, "get_styles", nil, nil))

	s.AddTool(mcp.NewTool("get_variable_defs",
		mcp.WithDescription("Get all local variable definitions: collections, modes, and values. Variables are Figma's design token system."),
	), makeHandler(node, "get_variable_defs", nil, nil))

	s.AddTool(mcp.NewTool("get_local_components",
		mcp.WithDescription("Get all components defined in the current Figma file."),
	), makeHandler(node, "get_local_components", nil, nil))

	s.AddTool(mcp.NewTool("get_annotations",
		mcp.WithDescription("Get all dev-mode annotations in the current document or on a specific node."),
		mcp.WithString("nodeId",
			mcp.Description("Optional node ID to filter annotations, colon format e.g. '4029:12345'"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := map[string]interface{}{}
		if id, ok := req.GetArguments()["nodeId"].(string); ok && id != "" {
			params["nodeId"] = id
		}
		resp, err := node.Send(ctx, "get_annotations", nil, params)
		return renderResponse(resp, err)
	})

	// ── Export ───────────────────────────────────────────────────────────

	s.AddTool(mcp.NewTool("get_screenshot",
		mcp.WithDescription("Export a screenshot of selected or specific nodes. Returns base64-encoded image data."),
		mcp.WithArray("nodeIds",
			mcp.Description("Optional node IDs to export, colon format. If empty, exports current selection."),
			mcp.WithStringItems(),
		),
		mcp.WithString("format",
			mcp.Description("Export format: PNG (default), SVG, JPG, or PDF"),
		),
		mcp.WithNumber("scale",
			mcp.Description("Export scale for raster formats (default 2)"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, _ := req.GetArguments()["nodeIds"].([]interface{})
		nodeIDs := toStringSlice(raw)
		params := map[string]interface{}{}
		if f, ok := req.GetArguments()["format"].(string); ok && f != "" {
			params["format"] = f
		}
		if s, ok := req.GetArguments()["scale"].(float64); ok && s > 0 {
			params["scale"] = s
		}
		resp, err := node.Send(ctx, "get_screenshot", nodeIDs, params)
		return renderResponse(resp, err)
	})

	s.AddTool(mcp.NewTool("save_screenshots",
		mcp.WithDescription("Export screenshots for multiple nodes and save them to the local filesystem. Returns metadata only (no base64)."),
		mcp.WithArray("items",
			mcp.Required(),
			mcp.Description("List of {nodeId, outputPath, format?, scale?} objects"),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nodeId":     map[string]any{"type": "string", "description": "Node ID in colon format e.g. '4029:12345'"},
					"outputPath": map[string]any{"type": "string", "description": "File path to write the image to"},
					"format":     map[string]any{"type": "string", "description": "Export format: PNG, SVG, JPG, or PDF"},
					"scale":      map[string]any{"type": "number", "description": "Export scale for raster formats"},
				},
				"required": []string{"nodeId", "outputPath"},
			}),
		),
		mcp.WithString("format",
			mcp.Description("Default export format: PNG (default), SVG, JPG, or PDF"),
		),
		mcp.WithNumber("scale",
			mcp.Description("Default export scale for raster formats (default 2)"),
		),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return executeSaveScreenshots(ctx, node, req)
	})
}
