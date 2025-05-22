package main

import (
	"encoding/json"
	"fmt"
	"log"       // Standard Go logger (will be used as fallback)
	"log/syslog" // Syslog package
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// --- Data Structures (same as before) ---
// Service represents an available service type
type Service struct {
	ServiceCode string   `json:"service_code"`
	ServiceName string   `json:"service_name"`
	Description string   `json:"description"`
	Metadata    bool     `json:"metadata"` // True if service_definition is available
	Type        string   `json:"type"`     // "realtime", "batch", "blackbox"
	Keywords    []string `json:"keywords"`
	Group       string   `json:"group"`
}

// ServiceRequestInput is the data expected when creating a new request
type ServiceRequestInput struct {
	APIKey         string  `json:"api_key"`
	ServiceCode    string  `json:"service_code"`
	Latitude       float64 `json:"lat,omitempty"`
	Longitude      float64 `json:"long,omitempty"`
	AddressString  string  `json:"address_string,omitempty"`
	Email          string  `json:"email,omitempty"`
	DeviceID       string  `json:"device_id,omitempty"`
	AccountID      string  `json:"account_id,omitempty"`
	FirstName      string  `json:"first_name,omitempty"`
	LastName       string  `json:"last_name,omitempty"`
	Phone          string  `json:"phone,omitempty"`
	Description    string  `json:"description,omitempty"`
	MediaURL       string  `json:"media_url,omitempty"`
}

// ServiceRequestOutput is the data returned for a service request
type ServiceRequestOutput struct {
	ServiceRequestID  string    `json:"service_request_id"`
	Status            string    `json:"status"` // "open", "closed"
	StatusNotes       string    `json:"status_notes,omitempty"`
	ServiceName       string    `json:"service_name"`
	ServiceCode       string    `json:"service_code"`
	Description       string    `json:"description,omitempty"`
	AgencyResponsible string    `json:"agency_responsible,omitempty"`
	ServiceNotice     string    `json:"service_notice,omitempty"`
	RequestedDatetime time.Time `json:"requested_datetime"`
	UpdatedDatetime   time.Time `json:"updated_datetime"`
	ExpectedDatetime  time.Time `json:"expected_datetime,omitempty"`
	Address           string    `json:"address,omitempty"`
	AddressID         string    `json:"address_id,omitempty"`
	Zipcode           string    `json:"zipcode,omitempty"`
	Latitude          float64   `json:"lat,omitempty"`
	Longitude         float64   `json:"long,omitempty"`
	MediaURL          string    `json:"media_url,omitempty"`
}


// --- In-Memory Storage (same as before) ---
var (
	services       []Service
	requests       = make(map[string]ServiceRequestOutput)
	requestCounter int
	mu             sync.Mutex
	validAPIKey    = "your-secret-api-key"
)

// --- Syslog Configuration ---
var (
	syslogWriter *syslog.Writer
	syslogTag    string = "open311-api" // Tag for syslog messages
)

// --- Initialization ---
func init() {
	// Populate some sample services
	services = []Service{
		{ServiceCode: "001", ServiceName: "Pothole Repair", Description: "Report a pothole in a city street.", Metadata: false, Type: "realtime", Keywords: []string{"pothole", "street", "road", "damage"}, Group: "Streets"},
		{ServiceCode: "002", ServiceName: "Graffiti Removal", Description: "Report graffiti on public or private property.", Metadata: false, Type: "realtime", Keywords: []string{"graffiti", "vandalism", "paint"}, Group: "Public Works"},
		{ServiceCode: "003", ServiceName: "Broken Streetlight", Description: "Report a streetlight that is out or malfunctioning.", Metadata: false, Type: "batch", Keywords: []string{"streetlight", "light", "outage"}, Group: "Utilities"},
	}
	requestCounter = 0

	// --- Initialize Syslog ---
	// You can make SYSLOG_ADDRESS configurable via environment variable
	syslogAddr := os.Getenv("SYSLOG_ADDRESS")
	if syslogAddr == "" {
		syslogAddr = "localhost:514" // Default syslog address (UDP)
	}

	var err error
	// syslog.LOG_LOCAL0 is an example facility. Choose one appropriate for your setup.
	// You can also use syslog.LOG_USER
	syslogWriter, err = syslog.Dial("udp", syslogAddr, syslog.LOG_LOCAL0|syslog.LOG_INFO, syslogTag)
	if err != nil {
		log.Printf("Failed to connect to syslog server at %s: %v. Logging to stdout/stderr.", syslogAddr, err)
		// syslogWriter will remain nil, and our logInfo/logErr functions will fall back
	} else {
		log.Printf("Successfully connected to syslog server at %s with tag '%s'", syslogAddr, syslogTag)
	}
}

// --- Custom Logging Functions ---

func logInfo(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	if syslogWriter != nil {
		err := syslogWriter.Info(message)
		if err != nil {
			log.Printf("Syslog (Info) write error: %v. Original message: %s", err, message)
		}
	} else {
		log.Printf("INFO: %s", message) // Fallback to standard log
	}
}

func logWarning(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	if syslogWriter != nil {
		err := syslogWriter.Warning(message)
		if err != nil {
			log.Printf("Syslog (Warning) write error: %v. Original message: %s", err, message)
		}
	} else {
		log.Printf("WARNING: %s", message) // Fallback to standard log
	}
}

func logError(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	if syslogWriter != nil {
		err := syslogWriter.Err(message) // Note: syslog.Writer.Err is for Error severity
		if err != nil {
			log.Printf("Syslog (Error) write error: %v. Original message: %s", err, message)
		}
	} else {
		log.Printf("ERROR: %s", message) // Fallback to standard log
	}
}

func logCritical(format string, v ...interface{}) {
	message := fmt.Sprintf(format, v...)
	if syslogWriter != nil {
		err := syslogWriter.Crit(message)
		if err != nil {
			log.Printf("Syslog (Critical) write error: %v. Original message: %s", err, message)
		}
	} else {
		log.Printf("CRITICAL: %s", message) // Fallback to standard log
	}
}

// --- Helper Functions (Modified to use new loggers) ---

func findServiceByCode(code string) *Service {
	for _, s := range services {
		if s.ServiceCode == code {
			return &s
		}
	}
	return nil
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	logWarning("Responding with error: Code %d, Message: %s", code, message) // Log warnings for client errors
	respondWithJSON(w, code, map[string]string{"error": message, "code": strconv.Itoa(code)})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		logError("Error marshalling JSON: %v", err) // Log errors for server issues
		http.Error(w, `{"error": "Internal Server Error marshalling JSON"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// --- Handlers (Modified to use new loggers) ---

// GET /services.json
func getServicesHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Received GET /services.json from %s", r.RemoteAddr)
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}
	respondWithJSON(w, http.StatusOK, services)
}

// POST /requests.json
func createRequestHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Received POST /requests.json from %s", r.RemoteAddr)
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var input ServiceRequestInput
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}
	defer r.Body.Close()

	if input.APIKey != validAPIKey {
		logWarning("Unauthorized API key attempt from %s for service %s", r.RemoteAddr, input.ServiceCode)
		respondWithError(w, http.StatusUnauthorized, "Invalid or missing API key.")
		return
	}
	if input.ServiceCode == "" {
		respondWithError(w, http.StatusBadRequest, "service_code is required.")
		return
	}
	service := findServiceByCode(input.ServiceCode)
	if service == nil {
		respondWithError(w, http.StatusBadRequest, "Invalid service_code.")
		return
	}
	if input.Latitude == 0 && input.Longitude == 0 && input.AddressString == "" {
		respondWithError(w, http.StatusBadRequest, "Either lat/long or address_string is required.")
		return
	}

	mu.Lock()
	requestCounter++
	serviceRequestID := fmt.Sprintf("SR%06d", requestCounter)
	now := time.Now().UTC()

	newRequest := ServiceRequestOutput{
		ServiceRequestID:  serviceRequestID,
		Status:            "open",
		StatusNotes:       "Request received and is pending review.",
		ServiceName:       service.ServiceName,
		ServiceCode:       input.ServiceCode,
		Description:       input.Description,
		AgencyResponsible: "City Department of " + service.Group,
		RequestedDatetime: now,
		UpdatedDatetime:   now,
		Address:           input.AddressString,
		Latitude:          input.Latitude,
		Longitude:         input.Longitude,
		MediaURL:          input.MediaURL,
	}
	requests[serviceRequestID] = newRequest
	mu.Unlock()

	responsePayload := []ServiceRequestOutput{newRequest}
	respondWithJSON(w, http.StatusCreated, responsePayload)
	logInfo("Created request: %s for service: %s by user: %s", serviceRequestID, input.ServiceCode, input.Email)
}

// GET /requests.json
// GET /requests/{service_request_id}.json
func getRequestsHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Received GET %s from %s", r.URL.Path, r.RemoteAddr)
	if r.Method != http.MethodGet {
		respondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	serviceRequestID := r.PathValue("service_request_id") // Go 1.22+

	mu.Lock()
	defer mu.Unlock()

	if serviceRequestID != "" {
		request, found := requests[serviceRequestID]
		if !found {
			respondWithError(w, http.StatusNotFound, "Service request not found: "+serviceRequestID)
			return
		}
		responsePayload := []ServiceRequestOutput{request}
		respondWithJSON(w, http.StatusOK, responsePayload)
	} else {
		filterServiceCode := r.URL.Query().Get("service_code")
		filterStatus := r.URL.Query().Get("status")
		
		var results []ServiceRequestOutput
		for _, req := range requests {
			match := true
			if filterServiceCode != "" && req.ServiceCode != filterServiceCode {
				match = false
			}
			if filterStatus != "" && req.Status != filterStatus {
				match = false
			}
			if match {
				results = append(results, req)
			}
		}
		respondWithJSON(w, http.StatusOK, results)
		logInfo("Listed %d requests. Filters: service_code=%s, status=%s", len(results), filterServiceCode, filterStatus)
	}
}

func main() {
	// Ensure syslogWriter is closed on exit (though for long-running servers,
	// explicit signal handling for graceful shutdown is better).
	if syslogWriter != nil {
		defer syslogWriter.Close()
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /services.json", getServicesHandler)
	mux.HandleFunc("POST /requests.json", createRequestHandler)
	mux.HandleFunc("GET /requests.json", getRequestsHandler)
	mux.HandleFunc("GET /requests/{service_request_id}.json", getRequestsHandler)

	port := ":8080"
	logInfo("Open311 API server starting on %s...", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		logCritical("Could not start server: %s\n", err) // Use critical for fatal server errors
		os.Exit(1) // Exit if server can't start
	}
}
