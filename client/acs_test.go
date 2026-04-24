package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewACS(t *testing.T) {
	acs := NewACS("http://localhost:7557", "admin", "pass")
	if acs.base != "http://localhost:7557" {
		t.Errorf("expected base http://localhost:7557, got %s", acs.base)
	}
	if acs.user != "admin" {
		t.Errorf("expected user admin, got %s", acs.user)
	}
	if acs.pass != "pass" {
		t.Errorf("expected pass pass, got %s", acs.pass)
	}
	if acs.hc == nil {
		t.Error("expected non-nil http client")
	}
}

func TestBuildURL(t *testing.T) {
	acs := NewACS("http://localhost:7557", "", "")

	tests := []struct {
		name     string
		path     string
		query    map[string]string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/devices/",
			query:    nil,
			expected: "http://localhost:7557/devices/",
		},
		{
			name:     "path with query",
			path:     "/devices/",
			query:    map[string]string{"query": `{"_id":"test"}`},
			expected: "http://localhost:7557/devices/?query=%7B%22_id%22%3A%22test%22%7D",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := make(map[string][]string)
			for k, v := range tt.query {
				q[k] = []string{v}
			}
			result := acs.buildURL(tt.path, q)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/devices/" {
			t.Errorf("expected /devices/, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if query != `{"_id":"device-001"}` {
			t.Errorf("unexpected query: %s", query)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{{"_id": "device-001"}})
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.GetDevice("device-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestGetDevice_WithAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"_id":"device-001"}]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "admin", "secret")
	body, err := acs.GetDevice("device-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestGetDevice_NoAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify no auth header when user is empty
		_, _, ok := r.BasicAuth()
		if ok {
			t.Error("did not expect basic auth to be set")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.GetDevice("test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDevice_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.GetDevice("device-001")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
	if err.Error() != "ACS error 500: internal error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestGetFileByName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files" {
			t.Errorf("expected /files, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if query != `{"filename":"firmware.bin"}` {
			t.Errorf("unexpected query: %s", query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"filename":"firmware.bin","_id":"abc123"}]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.GetFileByName("firmware.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestGetTasksForDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			t.Errorf("expected /tasks, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if query != `{"device":"dev-001"}` {
			t.Errorf("unexpected query: %s", query)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.GetTasksForDevice("dev-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "[]" {
		t.Errorf("expected [], got %s", string(body))
	}
}

func TestListDeviceSummaries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/devices" {
			t.Errorf("expected /devices, got %s", r.URL.Path)
		}
		limit := r.URL.Query().Get("limit")
		if limit != "100" {
			t.Errorf("expected limit 100, got %s", limit)
		}
		proj := r.URL.Query().Get("projection")
		if proj != "_id,summary.lastInform" {
			t.Errorf("unexpected projection: %s", proj)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"_id":"dev-001"}]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.ListDeviceSummaries(100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestListDeviceSummaries_NoLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		if limit != "" {
			t.Errorf("expected no limit param, got %s", limit)
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.ListDeviceSummaries(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRebootDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Verify the body contains reboot task
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "reboot" {
			t.Errorf("expected reboot task, got %v", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"_id":"task-001"}`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.RebootDevice("dev-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestDownloadFirmware(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "download" {
			t.Errorf("expected download task, got %v", body["name"])
		}
		if body["file"] != "file-001" {
			t.Errorf("expected file file-001, got %v", body["file"])
		}
		if body["filename"] != "firmware.bin" {
			t.Errorf("expected filename firmware.bin, got %v", body["filename"])
		}
		w.Write([]byte(`{"_id":"task-002"}`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.DownloadFirmware("dev-001", "file-001", "firmware.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestDownloadFirmware_NoFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["filename"]; ok {
			t.Error("did not expect filename key when empty")
		}
		w.Write([]byte(`{"_id":"task-003"}`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.DownloadFirmware("dev-001", "file-001", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRefreshParameter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "getParameterValues" {
			t.Errorf("expected getParameterValues task, got %v", body["name"])
		}
		params, ok := body["parameterNames"].([]any)
		if !ok || len(params) != 1 || params[0] != "Device.DeviceInfo.SerialNumber" {
			t.Errorf("unexpected parameterNames: %v", body["parameterNames"])
		}
		w.Write([]byte(`{"_id":"task-004"}`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.RefreshParameter("dev-001", "Device.DeviceInfo.SerialNumber")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestSetParameterValues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "setParameterValues" {
			t.Errorf("expected setParameterValues task, got %v", body["name"])
		}
		pv, ok := body["parameterValues"].([]any)
		if !ok || len(pv) != 1 {
			t.Errorf("unexpected parameterValues: %v", body["parameterValues"])
		}
		w.Write([]byte(`{"_id":"task-005"}`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	pv := json.RawMessage(`[["Device.WiFi.SSID","TestSSID","xsd:string"]]`)
	body, err := acs.SetParameterValues("dev-001", pv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestGetDeviceParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/devices/" {
			t.Errorf("expected /devices/, got %s", r.URL.Path)
		}
		proj := r.URL.Query().Get("projection")
		if proj != "Device.DeviceInfo.SoftwareVersion" {
			t.Errorf("unexpected projection: %s", proj)
		}
		w.Write([]byte(`[{"_id":"dev-001","Device.DeviceInfo.SoftwareVersion":{"_value":"1.0"}}]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.GetDeviceParameters("dev-001", "Device.DeviceInfo.SoftwareVersion")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestListPresets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/presets" {
			t.Errorf("expected /presets, got %s", r.URL.Path)
		}
		w.Write([]byte(`[{"_id":"preset-1","weight":0}]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.ListPresets()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestPutPreset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/presets/test-preset" {
			t.Errorf("expected /presets/test-preset, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.PutPreset("test-preset", []byte(`{"weight":0}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeletePreset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/presets/test-preset" {
			t.Errorf("expected /presets/test-preset, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.DeletePreset("test-preset")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListProvisions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/provisions" {
			t.Errorf("expected /provisions, got %s", r.URL.Path)
		}
		w.Write([]byte(`[{"_id":"prov-1","script":"log('hi');"}]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.ListProvisions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestPutProvision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/provisions/my-prov" {
			t.Errorf("expected /provisions/my-prov, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/javascript" {
			t.Errorf("expected application/javascript, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.PutProvision("my-prov", "log('hello');")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteProvision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/provisions/my-prov" {
			t.Errorf("expected /provisions/my-prov, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.DeleteProvision("my-prov")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetFaultsForDevice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/faults" {
			t.Errorf("expected /faults, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if query != `{"device":"dev-001"}` {
			t.Errorf("unexpected query: %s", query)
		}
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.GetFaultsForDevice("dev-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "[]" {
		t.Errorf("expected [], got %s", string(body))
	}
}

func TestSearchDevices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/devices" {
			t.Errorf("expected /devices, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if query != `{"_tags":"office"}` {
			t.Errorf("unexpected query: %s", query)
		}
		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("expected limit 10, got %s", limit)
		}
		w.Write([]byte(`[{"_id":"dev-001"}]`))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	body, err := acs.SearchDevices(`{"_tags":"office"}`, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestAddTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/devices/dev-001/tags/office" {
			t.Errorf("expected /devices/dev-001/tags/office, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.AddTag("dev-001", "office")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/devices/dev-001/tags/office" {
			t.Errorf("expected /devices/dev-001/tags/office, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.RemoveTag("dev-001", "office")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConnectionRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("connection_request") == "" && !r.URL.Query().Has("connection_request") {
			t.Error("expected connection_request query param")
		}
		if r.ContentLength > 0 {
			t.Error("expected empty body for connection request")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.ConnectionRequest("dev-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/tasks/task-001" {
			t.Errorf("expected /tasks/task-001, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.DeleteTask("task-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/tasks/task-001/retry" {
			t.Errorf("expected /tasks/task-001/retry, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.RetryTask("task-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteTask_DeviceInSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("device in session"))
	}))
	defer server.Close()

	acs := NewACS(server.URL, "", "")
	_, err := acs.DeleteTask("task-001")
	if err == nil {
		t.Fatal("expected error for 503 status")
	}
	if err.Error() != "ACS error 503: device in session" {
		t.Errorf("unexpected error: %s", err.Error())
	}
}
