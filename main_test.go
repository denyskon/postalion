// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	form_service "github.com/denyskon/postalion/form"
	"github.com/gorilla/pat"
	"github.com/sirupsen/logrus"
)

func TestFormHandlerSpamResponse(t *testing.T) {
	tests := []struct {
		name           string
		rejectRedirect string
		wantStatus     int
		wantLocation   string
		wantBody       string
	}{
		{
			name:       "default error response",
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   "Your submission was not accepted",
		},
		{
			name:           "custom rejection redirect",
			rejectRedirect: "https://example.com/message-not-sent",
			wantStatus:     http.StatusSeeOther,
			wantLocation:   "https://example.com/message-not-sent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origLog := log
			origForms := forms
			t.Cleanup(func() {
				log = origLog
				forms = origForms
			})

			ollama := httptest.NewServer(http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
				wr.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(wr).Encode(map[string]any{
					"message": map[string]string{
						"role":    "assistant",
						"content": `{"spam":true,"reason":"test spam"}`,
					},
				})
			}))
			defer ollama.Close()

			log = logrus.New().WithField("service", "test")
			forms = map[string]*form_service.Form{
				"test": {
					Type:     "mail",
					Redirect: "https://example.com/thanks",
					MailConfig: &form_service.MailConfig{
						Fields: []form_service.Field{
							{Name: "message", Label: "Message", Type: "string"},
						},
					},
					SpamCheck: &form_service.SpamCheckConfig{
						Endpoint:       ollama.URL,
						Model:          "test-model",
						RejectRedirect: tt.rejectRedirect,
					},
				},
			}

			router := pat.New()
			router.Post("/form/{name}", formHandler)

			req := httptest.NewRequest(http.MethodPost, "/form/test", strings.NewReader("message=spam"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			wr := httptest.NewRecorder()

			router.ServeHTTP(wr, req)

			if wr.Code != tt.wantStatus {
				t.Errorf("unexpected status %d, want %d", wr.Code, tt.wantStatus)
			}
			if location := wr.Header().Get("Location"); location != tt.wantLocation {
				t.Errorf("unexpected Location %q, want %q", location, tt.wantLocation)
			}
			if tt.wantBody != "" && !strings.Contains(wr.Body.String(), tt.wantBody) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBody, wr.Body.String())
			}
		})
	}
}
