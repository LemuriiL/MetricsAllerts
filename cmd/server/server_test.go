package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "valid gauge",
			method:         "POST",
			path:           "/update/gauge/test/42.5",
			contentType:    "text/plain",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid counter",
			method:         "POST",
			path:           "/update/counter/requests/10",
			contentType:    "text/plain",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty metric name",
			method:         "POST",
			path:           "/update/gauge//42.5",
			contentType:    "text/plain",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "invalid metric type",
			method:         "POST",
			path:           "/update/invalid/name/1",
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid number",
			method:         "POST",
			path:           "/update/gauge/test/abc",
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing Content-Type",
			method:         "POST",
			path:           "/update/gauge/test/1.0",
			contentType:    "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "wrong Content-Type",
			method:         "POST",
			path:           "/update/gauge/test/1.0",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET method",
			method:         "GET",
			path:           "/update/gauge/test/1.0",
			contentType:    "text/plain",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "double slash in path (should not redirect)",
			method:         "POST",
			path:           "/update/gauge//1",
			contentType:    "text/plain",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			updateHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			assert.NotContains(t, w.Header(), "Location")
		})
	}

	req := httptest.NewRequest("POST", "/update/gauge/UnitTest/99.9", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	updateHandler(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	if val, ok := store.GetGauge("UnitTest"); ok {
		assert.Equal(t, 99.9, val)
	} else {
		t.Error("metric UnitTest not saved")
	}
}
