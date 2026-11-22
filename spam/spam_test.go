// SPDX-License-Identifier: MIT
// Copyright (c) 2024 Denys Konovalov <kontakt@denyskon.de>

package spam

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	form_service "github.com/denyskon/postalion/form"
)

func TestCheckSpam(t *testing.T) {
	var received chatRequest

	server := httptest.NewServer(http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/chat" {
			t.Errorf("unexpected request path %q", req.URL.Path)
		}
		if err := json.NewDecoder(req.Body).Decode(&received); err != nil {
			t.Errorf("could not decode request: %v", err)
		}
		wr.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(wr).Encode(chatResponse{
			Message: chatMessage{
				Role:    "assistant",
				Content: `{"spam":true,"reason":"unsolicited advertising"}`,
			},
		})
	}))
	defer server.Close()

	form := &form_service.Form{
		Type: "mail",
		MailConfig: &form_service.MailConfig{
			Fields: []form_service.Field{
				{Name: "name", Label: "Name", Type: "string"},
				{Name: "message", Label: "Message", Type: "string"},
				{Name: "attachment", Label: "Attachment", Type: "file"},
			},
		},
		SpamCheck: &form_service.SpamCheckConfig{
			Endpoint: server.URL,
			Model:    "test-model",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/form/test", strings.NewReader("name=Bot&message=Buy+cheap+stuff&attachment=ignored"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	isSpam, reason, err := Check(form, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isSpam {
		t.Error("expected submission to be classified as spam")
	}
	if reason != "unsolicited advertising" {
		t.Errorf("unexpected reason %q", reason)
	}

	if received.Model != "test-model" {
		t.Errorf("unexpected model %q", received.Model)
	}
	if received.Stream {
		t.Error("expected stream to be disabled")
	}
	if received.Format == nil {
		t.Error("expected a structured response format")
	}
	if len(received.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(received.Messages))
	}

	content := received.Messages[1].Content
	if !strings.Contains(content, "Name: Bot") {
		t.Errorf("expected name field in prompt, got %q", content)
	}
	if !strings.Contains(content, "Message: Buy cheap stuff") {
		t.Errorf("expected message field in prompt, got %q", content)
	}
	if strings.Contains(content, "ignored") {
		t.Errorf("file fields should not be sent to the model, got %q", content)
	}
}

func TestSanitizeReason(t *testing.T) {
	got := sanitizeReason("ad\nspam\r\nInjected: 2026/01/01 INFO fake log")
	want := "ad spam  Injected: 2026/01/01 INFO fake log"
	if got != want {
		t.Errorf("unexpected sanitized reason %q, want %q", got, want)
	}
}
