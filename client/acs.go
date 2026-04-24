package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ACSClient struct {
	base string
	hc   *http.Client
	user string
	pass string
}

func NewACS(base, user, pass string) *ACSClient {
	return &ACSClient{
		base: base,
		hc:   &http.Client{},
		user: user,
		pass: pass,
	}
}

func (c *ACSClient) buildURL(path string, q url.Values) string {
	u, _ := url.Parse(c.base)
	u.Path = path
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *ACSClient) do(req *http.Request) ([]byte, error) {
	if c.user != "" {
		req.SetBasicAuth(c.user, c.pass)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ACS error %d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}

// Example high-level method – get device by ID
func (c *ACSClient) GetDevice(id string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", fmt.Sprintf(`{"_id":"%s"}`, id))
	url := c.buildURL("/devices/", q)
	req, _ := http.NewRequest("GET", url, nil)
	return c.do(req)
}

// GetFileByName returns the *metadata* (JSON) for a single file.
// If multiple files share the same filename it will return them all.
func (c *ACSClient) GetFileByName(fname string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", fmt.Sprintf(`{"filename":"%s"}`, fname))
	url := c.buildURL("/files", q)
	req, _ := http.NewRequest("GET", url, nil)
	return c.do(req)
}

// GetTasksForDevice returns ALL tasks (queued or executed) for a device.
func (c *ACSClient) GetTasksForDevice(devID string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", fmt.Sprintf(`{"device":"%s"}`, devID))
	url := c.buildURL("/tasks", q)
	req, _ := http.NewRequest("GET", url, nil)
	return c.do(req)
}

// ListDeviceSummaries returns _id + a few summary fields for ALL devices.
// Adds an optional limit parameter so we don’t DDoS ourselves.
func (c *ACSClient) ListDeviceSummaries(limit int) ([]byte, error) {
	q := url.Values{}
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
	// projection can be tuned; here we show only _id and summary.lastInform
	q.Add("projection", "_id,summary.lastInform")
	url := c.buildURL("/devices", q)
	req, _ := http.NewRequest("GET", url, nil)
	return c.do(req)
}

// private helper that actually POSTS the task
func (c *ACSClient) postTask(deviceID string, body any) ([]byte, error) {
	b, _ := json.Marshal(body)

	// we always ask for an immediate ConnectionRequest so that GenieACS
	// tries to do the action right away if the CPE is reachable.
	path := fmt.Sprintf("/devices/%s/tasks", url.PathEscape(deviceID))
	q := url.Values{"timeout": []string{"3000"}, "connection_request": []string{""}}
	endpoint := c.buildURL(path, q)

	req, _ := http.NewRequest("POST", endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

// Public helpers --------------------------------------------------------------

// RebootDevice enqueues a reboot task.
func (c *ACSClient) RebootDevice(deviceID string) ([]byte, error) {
	task := map[string]any{"name": "reboot"}
	return c.postTask(deviceID, task)
}

// DownloadFirmware triggers a file download. You *must* provide the fileID
// as stored in GridFS.  (filename is optional, but nice to have.)
func (c *ACSClient) DownloadFirmware(deviceID, fileID, filename string) ([]byte, error) {
	task := map[string]any{
		"name": "download",
		"file": fileID,
	}
	if filename != "" {
		task["filename"] = filename
	}
	return c.postTask(deviceID, task)
}

// RefreshParameter forces the CPE to send an updated value for paramName.
func (c *ACSClient) RefreshParameter(deviceID, paramName string) ([]byte, error) {
	task := map[string]any{
		"name":           "getParameterValues",
		"parameterNames": []string{paramName},
	}
	return c.postTask(deviceID, task)
}

// put sends a PUT request with the given body and content type.
func (c *ACSClient) put(path string, body []byte, contentType string) ([]byte, error) {
	endpoint := c.buildURL(path, nil)
	req, _ := http.NewRequest("PUT", endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	return c.do(req)
}

// deleteReq sends a DELETE request to the given path.
func (c *ACSClient) deleteReq(path string) ([]byte, error) {
	endpoint := c.buildURL(path, nil)
	req, _ := http.NewRequest("DELETE", endpoint, nil)
	return c.do(req)
}

// SetParameterValues queues a setParameterValues task on a device.
// parameterValues is a raw JSON array of [name, value, type] tuples.
func (c *ACSClient) SetParameterValues(deviceID string, parameterValues json.RawMessage) ([]byte, error) {
	task := map[string]any{
		"name":            "setParameterValues",
		"parameterValues": parameterValues,
	}
	return c.postTask(deviceID, task)
}

// GetDeviceParameters returns cached parameter values for a device using projection.
func (c *ACSClient) GetDeviceParameters(deviceID, projection string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", fmt.Sprintf(`{"_id":"%s"}`, deviceID))
	if projection != "" {
		q.Add("projection", projection)
	}
	u := c.buildURL("/devices/", q)
	req, _ := http.NewRequest("GET", u, nil)
	return c.do(req)
}

// ListPresets returns all presets as a JSON array.
func (c *ACSClient) ListPresets() ([]byte, error) {
	u := c.buildURL("/presets", nil)
	req, _ := http.NewRequest("GET", u, nil)
	return c.do(req)
}

// PutPreset creates or updates a preset. body is the full JSON preset document.
func (c *ACSClient) PutPreset(name string, body []byte) ([]byte, error) {
	return c.put(fmt.Sprintf("/presets/%s", url.PathEscape(name)), body, "application/json")
}

// DeletePreset removes a preset by name.
func (c *ACSClient) DeletePreset(name string) ([]byte, error) {
	return c.deleteReq(fmt.Sprintf("/presets/%s", url.PathEscape(name)))
}

// ListProvisions returns all provision scripts as a JSON array.
func (c *ACSClient) ListProvisions() ([]byte, error) {
	u := c.buildURL("/provisions", nil)
	req, _ := http.NewRequest("GET", u, nil)
	return c.do(req)
}

// PutProvision creates or updates a provision script. script is the JavaScript source.
func (c *ACSClient) PutProvision(name, script string) ([]byte, error) {
	return c.put(
		fmt.Sprintf("/provisions/%s", url.PathEscape(name)),
		[]byte(script),
		"application/javascript",
	)
}

// DeleteProvision removes a provision script by name.
func (c *ACSClient) DeleteProvision(name string) ([]byte, error) {
	return c.deleteReq(fmt.Sprintf("/provisions/%s", url.PathEscape(name)))
}

// GetFaultsForDevice returns fault records for a device.
func (c *ACSClient) GetFaultsForDevice(devID string) ([]byte, error) {
	q := url.Values{}
	q.Add("query", fmt.Sprintf(`{"device":"%s"}`, devID))
	u := c.buildURL("/faults", q)
	req, _ := http.NewRequest("GET", u, nil)
	return c.do(req)
}

// SearchDevices queries devices with a MongoDB-style filter string.
func (c *ACSClient) SearchDevices(query string, limit int) ([]byte, error) {
	q := url.Values{}
	q.Add("query", query)
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
	u := c.buildURL("/devices", q)
	req, _ := http.NewRequest("GET", u, nil)
	return c.do(req)
}

// AddTag adds a tag to a device.
func (c *ACSClient) AddTag(deviceID, tag string) ([]byte, error) {
	path := fmt.Sprintf("/devices/%s/tags/%s", url.PathEscape(deviceID), url.PathEscape(tag))
	endpoint := c.buildURL(path, nil)
	req, _ := http.NewRequest("POST", endpoint, nil)
	return c.do(req)
}

// RemoveTag removes a tag from a device.
func (c *ACSClient) RemoveTag(deviceID, tag string) ([]byte, error) {
	return c.deleteReq(fmt.Sprintf("/devices/%s/tags/%s", url.PathEscape(deviceID), url.PathEscape(tag)))
}

// ConnectionRequest sends a connection request to wake a CPE without queuing a task.
func (c *ACSClient) ConnectionRequest(deviceID string) ([]byte, error) {
	path := fmt.Sprintf("/devices/%s/tasks", url.PathEscape(deviceID))
	q := url.Values{"connection_request": {""}}
	endpoint := c.buildURL(path, q)
	req, _ := http.NewRequest("POST", endpoint, nil)
	return c.do(req)
}

// DeleteTask removes a pending task from the queue.
func (c *ACSClient) DeleteTask(taskID string) ([]byte, error) {
	return c.deleteReq(fmt.Sprintf("/tasks/%s", url.PathEscape(taskID)))
}

// RetryTask retries a faulted task.
func (c *ACSClient) RetryTask(taskID string) ([]byte, error) {
	endpoint := c.buildURL(fmt.Sprintf("/tasks/%s/retry", url.PathEscape(taskID)), nil)
	req, _ := http.NewRequest("POST", endpoint, nil)
	return c.do(req)
}
