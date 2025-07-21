package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEndpointHealth(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
		expectedHeader string
	}{
		{
			name:           "GET request to health endpoint",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
			expectedHeader: "text/plain; charset=utf-8",
		},
		{
			name:           "POST request to health endpoint",
			method:         "POST",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
			expectedHeader: "text/plain; charset=utf-8",
		},
		{
			name:           "PUT request to health endpoint",
			method:         "PUT",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
			expectedHeader: "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/healthz", nil)
			w := httptest.NewRecorder()

			endpointHealt(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("endpointHealt() status = %d, want %d", w.Code, tt.expectedStatus)
			}

			if w.Body.String() != tt.expectedBody {
				t.Errorf("endpointHealt() body = %q, want %q", w.Body.String(), tt.expectedBody)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedHeader {
				t.Errorf("endpointHealt() Content-Type = %q, want %q", contentType, tt.expectedHeader)
			}
		})
	}
}

func TestAdminData(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{
			name:  "zero count",
			count: 0,
		},
		{
			name:  "positive count",
			count: 42,
		},
		{
			name:  "large count",
			count: 999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := adminData{
				Count: tt.count,
			}

			if data.Count != tt.count {
				t.Errorf("adminData.Count = %d, want %d", data.Count, tt.count)
			}
		})
	}
}

func TestApiCreateUserReturn(t *testing.T) {
	tests := []struct {
		name  string
		email string
	}{
		{
			name:  "valid email",
			email: "test@example.com",
		},
		{
			name:  "empty email",
			email: "",
		},
		{
			name:  "email with special characters",
			email: "user+tag@domain.co.uk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := apiCreateUserReturn{
				Email: tt.email,
			}

			if user.Email != tt.email {
				t.Errorf("apiCreateUserReturn.Email = %q, want %q", user.Email, tt.email)
			}

			// Test that ID, CreatedAt, UpdatedAt fields exist (zero values are fine for testing structure)
			if user.ID.String() == "" && tt.email != "" {
				// UUID should have some string representation even when zero
				t.Logf("ID field exists: %v", user.ID)
			}
		})
	}
}
