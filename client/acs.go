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
