// Package client is a small HTTP client for the GenieACS northbound interface (NBI):
// it reads devices, files, tasks, faults, presets and provisions, and queues TR-069
// tasks such as reboots, firmware downloads and parameter changes.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DefaultMaxResponseBytes is the most the client reads from one NBI response
// unless SetMaxResponseBytes changes it.
const DefaultMaxResponseBytes = 4 << 20

// DefaultTimeout bounds one NBI request unless SetTimeout changes it.
const DefaultTimeout = 30 * time.Second

// maxErrorBody is how much of an NBI error body is passed on. The rest is
// dropped, so a large HTML error page never reaches the model.
const maxErrorBody = 512

// ErrResponseTooLarge is returned when an NBI response is larger than the
// configured maximum.
var ErrResponseTooLarge = errors.New("GenieACS answered with more data than ACS_MAX_RESPONSE_BYTES allows; narrow the query, request fewer fields with a projection, or lower the limit")

// ACSClient holds the NBI base URL, optional basic-auth credentials and an HTTP
// client. Every method returns the raw response body; an HTTP status of 400 or
// above is returned as an error carrying the start of that body.
type ACSClient struct {
	base    *url.URL
	hc      *http.Client
	user    string
	pass    string
	maxBody int64
}

// NewACS returns a client for the GenieACS NBI at base, such as
// "http://localhost:7557" or "https://acs.example.com/nbi". A path in base is
// kept as a prefix of every request. It sends basic auth only when user is not
// empty, and panics if base cannot be parsed as a URL.
func NewACS(base, user, pass string) *ACSClient {
	u, err := url.Parse(base)
	if err != nil {
		panic(fmt.Sprintf("invalid ACS base URL %q: %v", base, err))
	}
	return &ACSClient{
		base:    u,
		hc:      &http.Client{Timeout: DefaultTimeout},
		user:    user,
		pass:    pass,
		maxBody: DefaultMaxResponseBytes,
	}
}

// SetMaxResponseBytes caps how many bytes are read from one NBI response.
// Values below 1 keep the current limit.
func (c *ACSClient) SetMaxResponseBytes(n int64) {
	if n > 0 {
		c.maxBody = n
	}
}

// SetTimeout bounds one NBI request. Values below 1 keep the current timeout.
func (c *ACSClient) SetTimeout(d time.Duration) {
	if d > 0 {
		c.hc.Timeout = d
	}
}

// jsonQueryVal safely builds a JSON query object {"key":"value"} with proper escaping.
func jsonQueryVal(key, value string) string {
	v, _ := json.Marshal(value)
	return fmt.Sprintf(`{%q:%s}`, key, string(v))
}

// escPath joins path segments, escaping each one, into an escaped path that
// starts with "/". A device id such as "00259E-HG8245H%2D5-SN" keeps its "%"
// as data: GenieACS escapes every non-alphanumeric character of its ids, so the
// "%2D" is part of the id and must reach the NBI as "%252D".
func escPath(segments ...string) string {
	var b strings.Builder
	for _, s := range segments {
		b.WriteByte('/')
		b.WriteString(url.PathEscape(s))
	}
	return b.String()
}

// buildURL appends escapedPath to the base URL's path and sets the query.
// The escaped form goes in RawPath, so URL.String sends it as is instead of
// escaping its "%" a second time.
func (c *ACSClient) buildURL(escapedPath string, q url.Values) string {
	u := *c.base
	raw := strings.TrimRight(u.EscapedPath(), "/") + escapedPath
	p, err := url.PathUnescape(raw)
	if err != nil {
		p = raw
	}
	u.Path, u.RawPath = p, raw
	u.RawQuery = q.Encode()
	return u.String()
}

// response is one NBI answer: its status code and its body.
type response struct {
	status int
	body   []byte
}

func (c *ACSClient) send(ctx context.Context, method, endpoint string, body io.Reader, contentType string) (response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return response{}, fmt.Errorf("building the GenieACS request failed: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.user != "" {
		req.SetBasicAuth(c.user, c.pass)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return response{}, c.transportError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return response{}, fmt.Errorf("ACS error %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBody+1))
	if err != nil {
		return response{}, c.transportError(err)
	}
	if int64(len(b)) > c.maxBody {
		return response{}, ErrResponseTooLarge
	}
	return response{status: resp.StatusCode, body: b}, nil
}

// transportError turns a network failure into a message for the model that
// names no internal host, address or URL. The full error goes to the log.
func (c *ACSClient) transportError(err error) error {
	log.Printf("GenieACS NBI request failed: %v", err)
	var netErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return errors.New("the request was cancelled before GenieACS answered")
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return fmt.Errorf("GenieACS did not answer within %s", c.hc.Timeout)
	default:
		var opErr *net.OpError
		if errors.As(err, &opErr) {
			return errors.New("cannot reach the GenieACS NBI; check ACS_URL and that the NBI is running")
		}
		return errors.New("the request to the GenieACS NBI failed; the server log has the details")
	}
}

func (c *ACSClient) get(ctx context.Context, escapedPath string, q url.Values) ([]byte, error) {
	r, err := c.send(ctx, http.MethodGet, c.buildURL(escapedPath, q), nil, "")
	return r.body, err
}

// GetDevice returns the device document whose _id equals id, as a JSON array
// that is empty when no device matches.
func (c *ACSClient) GetDevice(ctx context.Context, id string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", jsonQueryVal("_id", id))
	return c.get(ctx, "/devices/", q)
}

// GetFileByName returns the *metadata* (JSON) for a single file.
// If multiple files share the same filename it will return them all.
func (c *ACSClient) GetFileByName(ctx context.Context, fname string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", jsonQueryVal("filename", fname))
	return c.get(ctx, "/files", q)
}

// GetTasksForDevice returns the tasks GenieACS still holds for a device: the
// ones waiting to run and the ones that faulted. GenieACS deletes a task once
// the device has run it.
func (c *ACSClient) GetTasksForDevice(ctx context.Context, devID string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", jsonQueryVal("device", devID))
	return c.get(ctx, "/tasks", q)
}

// DeviceSummaryProjection is the set of fields the device list returns: the
// id, the last inform time, the tags, and the identity GenieACS records for
// every device.
const DeviceSummaryProjection = "_id,_lastInform,_tags,_deviceId._Manufacturer,_deviceId._ProductClass,_deviceId._SerialNumber"

// ListDeviceSummaries returns the id, last inform time, tags, manufacturer,
// product class and serial number of up to limit devices (all when limit is 0).
func (c *ACSClient) ListDeviceSummaries(ctx context.Context, limit int) ([]byte, error) {
	q := url.Values{}
	if limit > 0 {
		q.Add("limit", strconv.Itoa(limit))
	}
	q.Add("projection", DeviceSummaryProjection)
	return c.get(ctx, "/devices", q)
}

// TaskResult is the NBI's answer to a queued task. GenieACS sends a connection
// request with every task and waits briefly: it answers 200 when the device ran
// the task in that session, and 202 when the task stays queued until the
// device next contacts the ACS.
type TaskResult struct {
	Status int
	Body   []byte
}

// Done reports whether the device ran the task before GenieACS answered.
func (r TaskResult) Done() bool { return r.Status == http.StatusOK }

// postTask queues body as a task on deviceID and asks GenieACS to run it now.
func (c *ACSClient) postTask(ctx context.Context, deviceID string, body any) (TaskResult, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return TaskResult{}, fmt.Errorf("failed to marshal task body: %w", err)
	}

	// we always ask for an immediate ConnectionRequest so that GenieACS
	// tries to do the action right away if the CPE is reachable.
	q := url.Values{"timeout": []string{"3000"}, "connection_request": []string{""}}
	endpoint := c.buildURL(escPath("devices", deviceID, "tasks"), q)
	r, err := c.send(ctx, http.MethodPost, endpoint, bytes.NewReader(b), "application/json")
	return TaskResult{Status: r.status, Body: r.body}, err
}

// Public helpers --------------------------------------------------------------

// RebootDevice enqueues a reboot task.
func (c *ACSClient) RebootDevice(ctx context.Context, deviceID string) (TaskResult, error) {
	return c.postTask(ctx, deviceID, map[string]any{"name": "reboot"})
}

// DownloadFirmware triggers a file download. You *must* provide the fileID
// as stored in GridFS.  (filename is optional, but nice to have.)
func (c *ACSClient) DownloadFirmware(ctx context.Context, deviceID, fileID, filename string) (TaskResult, error) {
	task := map[string]any{
		"name": "download",
		"file": fileID,
	}
	if filename != "" {
		task["targetFileName"] = filename
	}
	return c.postTask(ctx, deviceID, task)
}

// RefreshParameter forces the CPE to send an updated value for paramName.
func (c *ACSClient) RefreshParameter(ctx context.Context, deviceID, paramName string) (TaskResult, error) {
	task := map[string]any{
		"name":           "getParameterValues",
		"parameterNames": []string{paramName},
	}
	return c.postTask(ctx, deviceID, task)
}

// SetParameterValues queues a setParameterValues task on a device.
// parameterValues is a raw JSON array of [name, value, type] tuples.
func (c *ACSClient) SetParameterValues(ctx context.Context, deviceID string, parameterValues json.RawMessage) (TaskResult, error) {
	task := map[string]any{
		"name":            "setParameterValues",
		"parameterValues": parameterValues,
	}
	return c.postTask(ctx, deviceID, task)
}

// put sends a PUT request with the given body and content type.
func (c *ACSClient) put(ctx context.Context, escapedPath string, body []byte, contentType string) ([]byte, error) {
	r, err := c.send(ctx, http.MethodPut, c.buildURL(escapedPath, nil), bytes.NewReader(body), contentType)
	return r.body, err
}

// deleteReq sends a DELETE request to the given path.
func (c *ACSClient) deleteReq(ctx context.Context, escapedPath string) ([]byte, error) {
	r, err := c.send(ctx, http.MethodDelete, c.buildURL(escapedPath, nil), nil, "")
	return r.body, err
}

// post sends a POST with no body to the given path.
func (c *ACSClient) post(ctx context.Context, escapedPath string, q url.Values) ([]byte, error) {
	r, err := c.send(ctx, http.MethodPost, c.buildURL(escapedPath, q), nil, "")
	return r.body, err
}

// GetDeviceParameters returns cached parameter values for a device using projection.
func (c *ACSClient) GetDeviceParameters(ctx context.Context, deviceID, projection string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", jsonQueryVal("_id", deviceID))
	if projection != "" {
		q.Add("projection", projection)
	}
	return c.get(ctx, "/devices/", q)
}

// ListPresets returns all presets as a JSON array.
func (c *ACSClient) ListPresets(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/presets", nil)
}

// PutPreset creates or updates a preset. body is the full JSON preset document.
func (c *ACSClient) PutPreset(ctx context.Context, name string, body []byte) ([]byte, error) {
	return c.put(ctx, escPath("presets", name), body, "application/json")
}

// DeletePreset removes a preset by name.
func (c *ACSClient) DeletePreset(ctx context.Context, name string) ([]byte, error) {
	return c.deleteReq(ctx, escPath("presets", name))
}

// ListProvisions returns all provision scripts as a JSON array.
func (c *ACSClient) ListProvisions(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/provisions", nil)
}

// PutProvision creates or updates a provision script. script is the JavaScript source.
func (c *ACSClient) PutProvision(ctx context.Context, name, script string) ([]byte, error) {
	return c.put(ctx, escPath("provisions", name), []byte(script), "application/javascript")
}

// DeleteProvision removes a provision script by name.
func (c *ACSClient) DeleteProvision(ctx context.Context, name string) ([]byte, error) {
	return c.deleteReq(ctx, escPath("provisions", name))
}

// GetFaultsForDevice returns fault records for a device.
// Fault _id format is "deviceId:channel", so we match by regex prefix on _id.
// The id is quoted, so its characters only ever match themselves: an id such
// as ".*" cannot widen the query to every device's faults.
func (c *ACSClient) GetFaultsForDevice(ctx context.Context, devID string) ([]byte, error) {
	q := url.Values{}
	escaped, _ := json.Marshal("^" + regexp.QuoteMeta(devID) + ":")
	q.Add("query", fmt.Sprintf(`{"_id":{"$regex":%s}}`, string(escaped)))
	return c.get(ctx, "/faults", q)
}

// SearchOptions narrow a device search. Zero values leave the NBI defaults.
type SearchOptions struct {
	Limit      int    // most devices to return; 0 means no limit
	Skip       int    // devices to skip first, for paging
	Projection string // comma-separated fields to return; empty returns whole documents
	Sort       string // JSON object such as {"_lastInform":-1}
}

// SearchDevices queries devices with a MongoDB-style filter string.
func (c *ACSClient) SearchDevices(ctx context.Context, query string, opts SearchOptions) ([]byte, error) {
	q := url.Values{}
	q.Add("query", query)
	if opts.Limit > 0 {
		q.Add("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Skip > 0 {
		q.Add("skip", strconv.Itoa(opts.Skip))
	}
	if opts.Projection != "" {
		q.Add("projection", opts.Projection)
	}
	if opts.Sort != "" {
		q.Add("sort", opts.Sort)
	}
	return c.get(ctx, "/devices", q)
}

// AddTag adds a tag to a device.
func (c *ACSClient) AddTag(ctx context.Context, deviceID, tag string) ([]byte, error) {
	return c.post(ctx, escPath("devices", deviceID, "tags", tag), nil)
}

// RemoveTag removes a tag from a device.
func (c *ACSClient) RemoveTag(ctx context.Context, deviceID, tag string) ([]byte, error) {
	return c.deleteReq(ctx, escPath("devices", deviceID, "tags", tag))
}

// ConnectionRequest sends a connection request to wake a CPE without queuing a task.
func (c *ACSClient) ConnectionRequest(ctx context.Context, deviceID string) ([]byte, error) {
	return c.post(ctx, escPath("devices", deviceID, "tasks"), url.Values{"connection_request": {""}})
}

// DeleteTask removes a pending task from the queue.
func (c *ACSClient) DeleteTask(ctx context.Context, taskID string) ([]byte, error) {
	return c.deleteReq(ctx, escPath("tasks", taskID))
}

// RetryTask retries a faulted task.
func (c *ACSClient) RetryTask(ctx context.Context, taskID string) ([]byte, error) {
	return c.post(ctx, escPath("tasks", taskID, "retry"), nil)
}
