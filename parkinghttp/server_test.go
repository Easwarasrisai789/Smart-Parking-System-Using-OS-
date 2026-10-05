package parkinghttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	parkinglogic "smart-parking-kv/parking"
)

func TestParkingHTTPWorkflow(t *testing.T) {
	server, err := NewServer(parkinglogic.NewParkingLot())
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Handler()

	enter := request(t, handler, http.MethodPost, "/api/enter", `{"slotId":"slot-1","vehicleId":"car-1"}`)
	if enter.Code != http.StatusOK {
		t.Fatalf("enter status = %d, want %d: %s", enter.Code, http.StatusOK, enter.Body)
	}

	status := request(t, handler, http.MethodGet, "/api/slots/slot-1", "")
	var slot slotResponse
	if err := json.Unmarshal(status.Body.Bytes(), &slot); err != nil {
		t.Fatal(err)
	}
	if slot.SlotID != "slot-1" || slot.Status != "OCCUPIED" {
		t.Fatalf("unexpected slot response: %+v", slot)
	}

	conflict := request(t, handler, http.MethodPost, "/api/enter", `{"slotId":"slot-1","vehicleId":"car-2"}`)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("duplicate enter status = %d, want %d", conflict.Code, http.StatusConflict)
	}

	wrongVehicleExit := request(t, handler, http.MethodPost, "/api/exit", `{"slotId":"slot-1","vehicleId":"car-2"}`)
	if wrongVehicleExit.Code != http.StatusConflict {
		t.Fatalf("wrong vehicle exit status = %d, want %d: %s", wrongVehicleExit.Code, http.StatusConflict, wrongVehicleExit.Body)
	}

	status = request(t, handler, http.MethodGet, "/api/slots/slot-1", "")
	if err := json.Unmarshal(status.Body.Bytes(), &slot); err != nil {
		t.Fatal(err)
	}
	if slot.Status != "OCCUPIED" {
		t.Fatalf("slot status after wrong vehicle exit = %q, want OCCUPIED", slot.Status)
	}

	exit := request(t, handler, http.MethodPost, "/api/exit", `{"slotId":"slot-1","vehicleId":"car-1"}`)
	if exit.Code != http.StatusOK {
		t.Fatalf("exit status = %d, want %d: %s", exit.Code, http.StatusOK, exit.Body)
	}

	status = request(t, handler, http.MethodGet, "/api/slots/slot-1", "")
	if err := json.Unmarshal(status.Body.Bytes(), &slot); err != nil {
		t.Fatal(err)
	}
	if slot.Status != "AVAILABLE" {
		t.Fatalf("slot status after exit = %q, want AVAILABLE", slot.Status)
	}
}

func TestParkingHTTPRejectsInvalidRequest(t *testing.T) {
	server, err := NewServer(parkinglogic.NewParkingLot())
	if err != nil {
		t.Fatal(err)
	}

	for _, body := range []string{
		`{`,
		`{"slotId":"slot-1"}`,
		`{"slotId":"slot-1","vehicleId":"car-1","other":"not-allowed"}`,
	} {
		response := request(t, server.Handler(), http.MethodPost, "/api/enter", body)
		if response.Code != http.StatusBadRequest {
			t.Errorf("body %q status = %d, want %d", body, response.Code, http.StatusBadRequest)
		}
	}
}

func TestParkingHTTPHealthAndPage(t *testing.T) {
	server, err := NewServer(parkinglogic.NewParkingLot())
	if err != nil {
		t.Fatal(err)
	}

	health := request(t, server.Handler(), http.MethodGet, "/healthz", "")
	if health.Code != http.StatusOK || !bytes.Contains(health.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Fatalf("unexpected health response: %d %s", health.Code, health.Body)
	}
	page := request(t, server.Handler(), http.MethodGet, "/", "")
	if page.Code != http.StatusOK ||
		!bytes.Contains(page.Body.Bytes(), []byte("<title>Smart Parking</title>")) ||
		bytes.Contains(page.Body.Bytes(), []byte("Live Raft Monitoring")) {
		t.Fatalf("unexpected page response: %d", page.Code)
	}
}

func TestGrafanaDashboardProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/grafana/d/smart-parking-raft/smart-parking-raft" {
			t.Errorf("upstream path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("kiosk") != "" {
			t.Errorf("unexpected kiosk query value: %q", r.URL.Query().Get("kiosk"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "dashboard")
	}))
	defer upstream.Close()

	server, err := NewServer(parkinglogic.NewParkingLot())
	if err != nil {
		t.Fatal(err)
	}
	server.grafanaProxy, err = newGrafanaProxy(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}

	response := request(t, server.Handler(), http.MethodGet, "/grafana/d/smart-parking-raft/smart-parking-raft?orgId=1&kiosk", "")
	if response.Code != http.StatusOK || response.Body.String() != "dashboard" {
		t.Fatalf("Grafana proxy response = %d %q", response.Code, response.Body.String())
	}
}

func request(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}
