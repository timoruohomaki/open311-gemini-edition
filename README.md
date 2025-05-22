# Open311 Go API - Developer Reference Guide

This document provides a reference for developers working with or extending the Go-based Open311 API implementation.

## Table of Contents

1.  [Overview](#overview)
2.  [Prerequisites](#prerequisites)
3.  [Project Structure](#project-structure)
4.  [Setup and Installation](#setup-and-installation)
5.  [Configuration](#configuration)
    *   [API Key](#api-key)
    *   [Syslog Address](#syslog-address)
6.  [Running the Application](#running-the-application)
7.  [API Endpoints](#api-endpoints)
    *   [GET /services.json](#get-servicesjson)
    *   [POST /requests.json](#post-requestsjson)
    *   [GET /requests.json](#get-requestsjson-list)
    *   [GET /requests/{service_request_id}.json](#get-requestsservice_request_idjson-specific)
8.  [Data Structures](#data-structures)
9.  [Logging](#logging)
10. [Testing with cURL](#testing-with-curl)
11. [Dependencies](#dependencies)
12. [Future Enhancements & Contributing](#future-enhancements--contributing)

## 1. Overview

This project implements a subset of the [Open311 GeoReport v2](http://wiki.open311.org/GeoReport_v2/) specification using Go's standard `net/http` package. It provides endpoints for discovering service types and creating/querying service requests.

Currently, data is stored in-memory and logging is supported via UDP to a syslog server (with a fallback to stdout/stderr).

## 2. Prerequisites

*   **Go:** Version 1.22 or later (for native path parameter support like `/requests/{id}.json`).
    *   If using an older Go version, the routing for specific request IDs will need to be adapted (e.g., using a third-party router like `gorilla/mux` or manual path parsing).
*   **(Optional) Syslog Server:** A syslog server (e.g., rsyslog, syslog-ng) configured to listen on a UDP port (default: `localhost:514`) if you want to use syslog.

## 3. Project Structure

The application consists of a single Go file (`open311_api.go`):

*   **Data Structures:** Structs defining `Service`, `ServiceRequestInput`, `ServiceRequestOutput`.
*   **In-Memory Storage:** Global variables for `services`, `requests`, `requestCounter`, and a `sync.Mutex` for thread safety.
*   **Syslog Configuration:** Variables and initialization for the syslog writer.
*   **Custom Logging Functions:** `logInfo`, `logWarning`, `logError`, `logCritical` that wrap the syslog writer or fall back to standard log.
*   **Helper Functions:** `findServiceByCode`, `respondWithError`, `respondWithJSON`.
*   **HTTP Handlers:**
    *   `getServicesHandler`
    *   `createRequestHandler`
    *   `getRequestsHandler` (handles both listing and specific request retrieval)
*   **`main()` function:** Initializes the HTTP server and routes.
*   **`init()` function:** Initializes sample services and the syslog connection.

## 4. Setup and Installation

1.  **Clone the Repository (if applicable):**
    ```bash
    git clone <repository-url>
    cd <repository-directory>
    ```
    If you just have the `open311_api.go` file, place it in a new directory.

2.  **Tidy Dependencies (good practice, though not strictly needed for this example as it only uses standard library):**
    ```bash
    go mod init <your-module-name> # e.g., go mod init open311api
    go mod tidy
    ```

## 5. Configuration

Configuration is primarily handled via hardcoded values or environment variables.

### API Key

*   **Location:** `validAPIKey` variable in `open311_api.go`.
*   **Default:** `"your-secret-api-key"`
*   **Recommendation:** For production, this should be managed securely (e.g., environment variables, secrets manager) and not hardcoded. The current implementation uses a single static key.

### Syslog Address

*   **Environment Variable:** `SYSLOG_ADDRESS`
*   **Format:** `hostname:port` or `ip_address:port`
*   **Default:** `localhost:514` (if `SYSLOG_ADDRESS` is not set)
*   **Example:**
    ```bash
    export SYSLOG_ADDRESS="syslog.example.com:514"
    ```

## 6. Running the Application

Execute the Go program:

```bash
go run open311_api.go
```

The server will start, and by default, listen on port `8080`. Log messages will indicate the status:

```
INFO: Successfully connected to syslog server at localhost:514 with tag 'open311-api'
INFO: Open311 API server starting on :8080...
```
(Or a fallback message if syslog connection fails).

To run with a custom syslog server address:
```bash
SYSLOG_ADDRESS="192.168.1.100:514" go run open311_api.go
```

## 7. API Endpoints

All responses are in JSON format.

### GET /services.json

*   **Description:** Lists all available service types that can be reported.
*   **Method:** `GET`
*   **Request Body:** None
*   **Success Response (200 OK):** An array of service objects.
    ```json
    [
      {
        "service_code": "001",
        "service_name": "Pothole Repair",
        "description": "Report a pothole in a city street.",
        "metadata": false,
        "type": "realtime",
        "keywords": ["pothole", "street", "road", "damage"],
        "group": "Streets"
      },
      {
        "service_code": "002",
        "service_name": "Graffiti Removal",
        // ... other fields
      }
    ]
    ```
*   **Error Responses:**
    *   `405 Method Not Allowed`

### POST /requests.json

*   **Description:** Creates a new service request.
*   **Method:** `POST`
*   **Request Headers:**
    *   `Content-Type: application/json`
*   **Request Body:** A JSON object with service request details.
    ```json
    {
      "api_key": "your-secret-api-key",
      "service_code": "001",
      "lat": 34.0522,
      "long": -118.2437,
      "address_string": "123 Main St, Anytown",
      "email": "citizen@example.com",
      "first_name": "John",
      "last_name": "Doe",
      "phone": "555-123-4567",
      "description": "Large pothole in front of the bakery.",
      "media_url": "http://example.com/pothole.jpg"
    }
    ```
    *   **Required:** `api_key`, `service_code`, and either (`lat`, `long`) or `address_string`.
*   **Success Response (201 Created):** An array containing a single service request object that was created.
    ```json
    [
      {
        "service_request_id": "SR000001",
        "status": "open",
        "status_notes": "Request received and is pending review.",
        "service_name": "Pothole Repair",
        "service_code": "001",
        "description": "Large pothole in front of the bakery.",
        "agency_responsible": "City Department of Streets",
        "requested_datetime": "2023-10-27T10:30:00Z",
        "updated_datetime": "2023-10-27T10:30:00Z",
        "address": "123 Main St, Anytown",
        "lat": 34.0522,
        "long": -118.2437,
        "media_url": "http://example.com/pothole.jpg"
      }
    ]
    ```
*   **Error Responses:**
    *   `400 Bad Request`: Invalid payload, missing required fields, invalid `service_code`.
    *   `401 Unauthorized`: Invalid or missing `api_key`.
    *   `405 Method Not Allowed`

### GET /requests.json (List)

*   **Description:** Lists existing service requests. Can be filtered using query parameters.
*   **Method:** `GET`
*   **Query Parameters (Optional):**
    *   `service_code`: Filter by service code (e.g., `?service_code=001`).
    *   `status`: Filter by status (e.g., `?status=open`).
    *   *(Other Open311 standard filters like `start_date`, `end_date`, `service_request_id` (for multiple IDs) are not fully implemented in this example but can be added).*
*   **Request Body:** None
*   **Success Response (200 OK):** An array of service request objects matching the criteria.
    ```json
    [
      {
        "service_request_id": "SR000001",
        // ... other fields
      },
      {
        "service_request_id": "SR000002",
        // ... other fields
      }
    ]
    ```
*   **Error Responses:**
    *   `405 Method Not Allowed`

### GET /requests/{service_request_id}.json (Specific)

*   **Description:** Retrieves details of a specific service request.
*   **Method:** `GET`
*   **Path Parameter:**
    *   `service_request_id`: The unique ID of the service request (e.g., `SR000001`).
*   **Request Body:** None
*   **Success Response (200 OK):** An array containing the single requested service request object.
    ```json
    [
      {
        "service_request_id": "SR000001",
        "status": "open",
        "status_notes": "Request received and is pending review.",
        "service_name": "Pothole Repair",
        "service_code": "001",
        // ... other fields
      }
    ]
    ```
*   **Error Responses:**
    *   `404 Not Found`: If the `service_request_id` does not exist.
    *   `405 Method Not Allowed`

## 8. Data Structures

The core data structures are defined in `open311_api.go`:

*   `Service`: Represents an available service type.
    *   Fields: `ServiceCode`, `ServiceName`, `Description`, `Metadata`, `Type`, `Keywords`, `Group`.
*   `ServiceRequestInput`: Data structure for creating a new service request.
    *   Fields: `APIKey`, `ServiceCode`, `Latitude`, `Longitude`, `AddressString`, `Email`, `DeviceID`, `AccountID`, `FirstName`, `LastName`, `Phone`, `Description`, `MediaURL`.
*   `ServiceRequestOutput`: Data structure for representing an existing service request.
    *   Fields: `ServiceRequestID`, `Status`, `StatusNotes`, `ServiceName`, `ServiceCode`, `Description`, `AgencyResponsible`, `ServiceNotice`, `RequestedDatetime`, `UpdatedDatetime`, `ExpectedDatetime`, `Address`, `AddressID`, `Zipcode`, `Latitude`, `Longitude`, `MediaURL`.

Refer to the source code for exact field types and JSON tags.

## 9. Logging

The application uses Go's standard `log/syslog` package to send logs over UDP to a configured syslog server.

*   **Syslog Server:** Configured via `SYSLOG_ADDRESS` environment variable (defaults to `localhost:514`).
*   **Facility:** `syslog.LOG_LOCAL0` (can be changed in code).
*   **Tag:** `open311-api` (prepended to log messages).
*   **Fallback:** If the connection to the syslog server fails, logs are written to standard output/error prefixed with `INFO:`, `WARNING:`, `ERROR:`, or `CRITICAL:`.
*   **Custom Log Functions:**
    *   `logInfo(format string, v ...interface{})`
    *   `logWarning(format string, v ...interface{})`
    *   `logError(format string, v ...interface{})`
    *   `logCritical(format string, v ...interface{})`

**Example Syslog Setup (rsyslog on Linux):**

1.  Edit `/etc/rsyslog.conf` or add a file in `/etc/rsyslog.d/`:
    ```
    # provides UDP syslog reception
    module(load="imudp")
    input(type="imudp" port="514")

    # Optional: Log messages from local0 to a specific file
    local0.*    /var/log/open311-api.log
    ```
2.  Restart rsyslog: `sudo systemctl restart rsyslog`
3.  Monitor the log file: `tail -f /var/log/open311-api.log` (or `/var/log/syslog` / `/var/log/messages`).

## 10. Testing with cURL

Replace `your-secret-api-key` with the actual key if you changed it.

*   **Get Services:**
    ```bash
    curl http://localhost:8080/services.json
    ```

*   **Create a New Request:**
    ```bash
    curl -X POST -H "Content-Type: application/json" -d '{
      "api_key": "your-secret-api-key",
      "service_code": "001",
      "lat": 34.0522,
      "long": -118.2437,
      "description": "Pothole at intersection X and Y."
    }' http://localhost:8080/requests.json
    ```
    *(Note the `service_request_id` returned, e.g., `SR000001`)*

*   **Get a Specific Request (use ID from previous step):**
    ```bash
    curl http://localhost:8080/requests/SR000001.json
    ```

*   **List All Requests:**
    ```bash
    curl http://localhost:8080/requests.json
    ```

*   **List Requests Filtered by Service Code:**
    ```bash
    curl "http://localhost:8080/requests.json?service_code=001"
    ```

*   **List Requests Filtered by Status:**
    ```bash
    curl "http://localhost:8080/requests.json?status=open"
    ```

## 11. Dependencies

This project currently uses only Go's standard library packages:
*   `encoding/json`
*   `fmt`
*   `log`
*   `log/syslog`
*   `net/http`
*   `os`
*   `strconv`
*   `sync`
*   `time`

No external Go modules are required.

## 12. Future Enhancements & Contributing

This is a basic implementation. Potential areas for improvement include:

*   **Database Integration:** Replace in-memory storage with a persistent database (PostgreSQL, MySQL, MongoDB, etc.).
*   **Robust Authentication/Authorization:** Implement proper API key management, OAuth2, or other secure methods.
*   **Input Validation:** More thorough validation of all input fields.
*   **Error Handling:** Standardized error response formats.
*   **Configuration Management:** Use a dedicated configuration library or Viper.
*   **Service Definitions (`GET /services/{service_code}.json`):** Implement this endpoint if `metadata` is true for a service.
*   **XML Support:** Add content negotiation or separate `.xml` endpoints.
*   **Unit and Integration Tests.**
*   **Rate Limiting.**
*   **CORS Configuration.**
*   **Graceful Shutdown.**
*   **Full implementation of Open311 GeoReport v2 query parameters.**

Contributions are welcome! Please feel free to fork the repository, make changes, and submit a pull request.
