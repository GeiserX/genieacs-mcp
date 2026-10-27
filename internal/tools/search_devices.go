package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Search limits: the default page and the largest one a call may ask for.
const (
	defaultSearchLimit = 50
	maxSearchLimit     = 200
	// maxRegexLength caps each $regex pattern, so a call cannot hand the
	// ACS's MongoDB a huge or pathological pattern to scan every device with.
	maxRegexLength = 256
)

// forbiddenOperators run code or whole-collection expressions on the ACS's
// MongoDB; a device search never needs them.
var forbiddenOperators = map[string]bool{"$where": true, "$function": true, "$accumulator": true, "$expr": true}

// SearchOutput is the structured result of search_devices.
type SearchOutput struct {
	Count   int   `json:"count" jsonschema_description:"Devices in this page."`
	Limit   int   `json:"limit" jsonschema_description:"The page size used."`
	Skip    int   `json:"skip" jsonschema_description:"Devices skipped before this page."`
	More    bool  `json:"more" jsonschema_description:"true when the page is full, so more devices may match: call again with skip = skip + count."`
	Devices []any `json:"devices" jsonschema_description:"The matching device documents, with the fields the projection asked for."`
}

// NewSearchDevices returns the search_devices tool, which finds devices with a
// MongoDB-style query and returns one page of them.
func NewSearchDevices(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("search_devices", hints("Search devices", true, false, true),
		mcp.WithDescription(
			"Find CPE devices in GenieACS with a MongoDB-style filter: by tag, manufacturer, model, firmware version, last inform "+
				"time or any TR-069 parameter. It only reads. Examples: {\"_tags\":\"office\"}; "+
				"{\"InternetGatewayDevice.DeviceInfo.Manufacturer\":\"Huawei\"} (GenieACS adds ._value to parameter paths); "+
				"{\"_lastInform\":{\"$lt\":\"2026-01-01T00:00:00Z\"}} for devices silent since that date; "+
				"{\"_id\":{\"$regex\":\"^202BC1-\"}}. Query operators such as $eq, $ne, $gt, $lt, $in, $nin, $exists, $regex, "+
				"$and and $or work; $where, $function, $accumulator and $expr are refused, and a $regex pattern may be at most 256 characters. "+
				"By default each device comes back as a summary (id, last inform, tags, manufacturer, product class, serial); "+
				"ask for other fields with projection. Results come in pages of limit (default 50, at most 200); page with skip.",
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.MinLength(2),
			mcp.Description("The filter, as a JSON object string. Example: {\"_tags\":\"office\"} or {\"_id\":\"202BC1-BM632w-000001\"}."),
		),
		mcp.WithInteger("limit",
			mcp.Min(1),
			mcp.Max(maxSearchLimit),
			mcp.DefaultNumber(defaultSearchLimit),
			mcp.Description("Most devices to return in this page, 1 to 200. Default 50."),
		),
		mcp.WithInteger("skip",
			mcp.Min(0),
			mcp.DefaultNumber(0),
			mcp.Description("Matching devices to skip before this page, for paging. Default 0."),
		),
		mcp.WithString("projection",
			mcp.Description("Comma-separated fields to return per device, e.g. \"_id,InternetGatewayDevice.DeviceInfo.SoftwareVersion\". "+
				"Default: the summary fields. \"all\" returns whole device documents, which can be large (tens of KB each)."),
		),
		mcp.WithString("sort",
			mcp.Description("Sort order as a JSON object string: 1 ascending, -1 descending. Example: {\"_lastInform\":-1} for the most recent first."),
		),
		mcp.WithOutputSchema[SearchOutput](),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var filter map[string]any
		if err := json.Unmarshal([]byte(query), &filter); err != nil {
			return mcp.NewToolResultError("query must be valid JSON: a filter object such as {\"_tags\":\"office\"}"), nil
		}
		if op := forbiddenOperator(filter); op != "" {
			return mcp.NewToolResultError(fmt.Sprintf("query uses %s, which runs code on the ACS database; use ordinary query operators", op)), nil
		}
		if longRegex(filter) {
			return mcp.NewToolResultError(fmt.Sprintf("a $regex pattern in query is longer than %d characters; use a shorter pattern", maxRegexLength)), nil
		}

		opts := client.SearchOptions{
			Limit:      req.GetInt("limit", defaultSearchLimit),
			Skip:       max(req.GetInt("skip", 0), 0),
			Projection: client.DeviceSummaryProjection,
		}
		if opts.Limit < 1 {
			opts.Limit = defaultSearchLimit
		}
		opts.Limit = min(opts.Limit, maxSearchLimit)
		switch p := strings.TrimSpace(req.GetString("projection", "")); p {
		case "":
		case "all":
			opts.Projection = ""
		default:
			opts.Projection = p
		}
		if s := strings.TrimSpace(req.GetString("sort", "")); s != "" {
			var sortDoc map[string]any
			if err := json.Unmarshal([]byte(s), &sortDoc); err != nil {
				return mcp.NewToolResultError("sort must be a JSON object such as {\"_lastInform\":-1}"), nil
			}
			opts.Sort = s
		}

		resp, err := acs.SearchDevices(ctx, query, opts)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var devices []any
		if err := json.Unmarshal(resp, &devices); err != nil {
			return mcp.NewToolResultError("GenieACS returned something other than a device list"), nil
		}
		if devices == nil {
			devices = []any{}
		}
		out := SearchOutput{Count: len(devices), Limit: opts.Limit, Skip: opts.Skip, More: len(devices) == opts.Limit, Devices: devices}
		text := fmt.Sprintf("Search results (limit %d, skip %d, %d devices", out.Limit, out.Skip, out.Count)
		if out.More {
			text += fmt.Sprintf("; the page is full, call again with skip=%d for more", out.Skip+out.Count)
		}
		return mcp.NewToolResultStructured(out, text+"): "+string(resp)), nil
	}
	return tool, handler
}

// forbiddenOperator returns the first forbidden operator used anywhere in v.
func forbiddenOperator(v any) string {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if forbiddenOperators[k] {
				return k
			}
			if op := forbiddenOperator(child); op != "" {
				return op
			}
		}
	case []any:
		for _, child := range x {
			if op := forbiddenOperator(child); op != "" {
				return op
			}
		}
	}
	return ""
}

// longRegex reports whether any $regex pattern in v is longer than maxRegexLength.
func longRegex(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if p, ok := child.(string); ok && k == "$regex" && len(p) > maxRegexLength {
				return true
			}
			if longRegex(child) {
				return true
			}
		}
	case []any:
		for _, child := range x {
			if longRegex(child) {
				return true
			}
		}
	}
	return false
}
