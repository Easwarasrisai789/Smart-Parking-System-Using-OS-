package parkinghttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	parkinglogic "smart-parking-kv/parking"
)

type Server struct {
	parkingLot   *parkinglogic.ParkingLot
	grafanaProxy *httputil.ReverseProxy
}

type parkingRequest struct {
	SlotID    string `json:"slotId"`
	VehicleID string `json:"vehicleId"`
}

type parkingResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type slotResponse struct {
	SlotID string `json:"slotId"`
	Status string `json:"status"`
}

func NewServer(parkingLot *parkinglogic.ParkingLot) (*Server, error) {
	if parkingLot == nil {
		return nil, errors.New("parking lot is required")
	}
	grafanaURL := os.Getenv("GRAFANA_URL")
	if grafanaURL == "" {
		grafanaURL = "http://grafana.parking.svc.cluster.local:3000"
	}
	grafanaProxy, err := newGrafanaProxy(grafanaURL)
	if err != nil {
		return nil, err
	}
	return &Server{parkingLot: parkingLot, grafanaProxy: grafanaProxy}, nil
}

func newGrafanaProxy(rawURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(rawURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("invalid Grafana URL %q", rawURL)
	}
	return httputil.NewSingleHostReverseProxy(target), nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /grafana/", s.grafanaProxy)
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /api/slots/{slotID}", s.status)
	mux.HandleFunc("POST /api/enter", s.enter)
	mux.HandleFunc("POST /api/exit", s.exit)
	return mux
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, indexHTML)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	slotID := strings.TrimSpace(r.PathValue("slotID"))
	if slotID == "" {
		writeError(w, http.StatusBadRequest, "slotId is required")
		return
	}
	writeJSON(w, http.StatusOK, slotResponse{
		SlotID: slotID,
		Status: s.parkingLot.Status(slotID),
	})
}

func (s *Server) enter(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRequest(w, r)
	if !ok {
		return
	}
	if err := s.parkingLot.Enter(request.SlotID, request.VehicleID); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, parkingResponse{
		Success: true,
		Message: fmt.Sprintf("Vehicle %s entered slot %s", request.VehicleID, request.SlotID),
	})
}

func (s *Server) exit(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeRequest(w, r)
	if !ok {
		return
	}
	if err := s.parkingLot.Exit(request.SlotID, request.VehicleID); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, parkingResponse{
		Success: true,
		Message: fmt.Sprintf("Vehicle %s exited slot %s", request.VehicleID, request.SlotID),
	})
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (parkingRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request parkingRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON with slotId and vehicleId")
		return parkingRequest{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return parkingRequest{}, false
	}
	request.SlotID = strings.TrimSpace(request.SlotID)
	request.VehicleID = strings.TrimSpace(request.VehicleID)
	if request.SlotID == "" || request.VehicleID == "" {
		writeError(w, http.StatusBadRequest, "slotId and vehicleId are required")
		return parkingRequest{}, false
	}
	return request, true
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, parkingResponse{Success: false, Message: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Smart Parking</title>
  <style>
    :root{font:16px system-ui,sans-serif;color:#172033;background:#f2f5fa}
    body{max-width:760px;margin:4rem auto;padding:0 1rem}
    main{background:#fff;border-radius:16px;padding:2rem;box-shadow:0 12px 36px #14213d18}
    h1{margin-top:0}.grid{display:grid;grid-template-columns:1fr 1fr;gap:1rem}
    label{display:grid;gap:.4rem}input,button{font:inherit;padding:.7rem;border-radius:8px}
    input{border:1px solid #b9c4d4}button{border:0;background:#1459c7;color:#fff;cursor:pointer}
    button.secondary{background:#34435c}.actions{display:flex;gap:.7rem;margin-top:1rem}
    #result{min-height:1.5rem;margin-top:1.25rem;font-weight:600}
    @media(max-width:520px){.grid{grid-template-columns:1fr}}
  </style>
</head>
<body><main>
  <h1>Smart Parking</h1>
  <p>Check a slot or record a vehicle entry or exit.</p>
  <form id="parking-form">
    <div class="grid">
      <label>Parking slot<input id="slot" required value="slot-1"></label>
      <label>Vehicle ID<input id="vehicle" required value="vehicle-1"></label>
    </div>
    <div class="actions">
      <button type="button" id="status" class="secondary">Check status</button>
      <button type="button" id="enter">Enter</button>
      <button type="button" id="exit">Exit</button>
    </div>
  </form>
  <div id="result" role="status" aria-live="polite"></div>
</main>
<script>
const result=document.querySelector("#result");
const slot=()=>document.querySelector("#slot").value.trim();
const vehicle=()=>document.querySelector("#vehicle").value.trim();
async function call(path,method="GET"){
  result.textContent="Working...";
  try{
    const response=await fetch(path,{method,headers:{"Content-Type":"application/json"},
      body:method==="GET"?undefined:JSON.stringify({slotId:slot(),vehicleId:vehicle()})});
    const data=await response.json();
    result.textContent=data.message||("Slot "+data.slotId+" is "+data.status);
  }catch(error){result.textContent="Request failed: "+error.message;}
}
document.querySelector("#status").addEventListener("click",()=>call("/api/slots/"+encodeURIComponent(slot())));
document.querySelector("#enter").addEventListener("click",()=>call("/api/enter","POST"));
document.querySelector("#exit").addEventListener("click",()=>call("/api/exit","POST"));
</script></body></html>`
