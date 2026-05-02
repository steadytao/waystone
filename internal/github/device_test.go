// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestDeviceCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("client_id"); got != "client-id" {
			t.Fatalf("client_id = %q, want client-id", got)
		}
		_ = json.NewEncoder(w).Encode(DeviceCode{
			DeviceCode:      "device",
			UserCode:        "USER-CODE",
			VerificationURI: "https://github.com/login/device",
			ExpiresIn:       900,
			Interval:        1,
		})
	}))
	defer server.Close()

	code, err := requestDeviceCode(context.Background(), server.Client(), server.URL, "client-id", "")
	if err != nil {
		t.Fatalf("requestDeviceCode returned error: %v", err)
	}
	if code.UserCode != "USER-CODE" {
		t.Fatalf("user code = %q, want USER-CODE", code.UserCode)
	}
}

func TestPollDeviceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != deviceGrantType {
			t.Fatalf("grant_type = %q, want %q", got, deviceGrantType)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "token",
			"token_type":   "bearer",
			"scope":        "repo",
		})
	}))
	defer server.Close()

	code := DeviceCode{DeviceCode: "device", ExpiresIn: 5, Interval: 1}
	token, err := pollDeviceToken(context.Background(), server.Client(), server.URL, "client-id", code)
	if err != nil {
		t.Fatalf("pollDeviceToken returned error: %v", err)
	}
	if token.AccessToken != "token" {
		t.Fatalf("access token = %q, want token", token.AccessToken)
	}
}

func TestPollDeviceTokenSlowDown(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":    "slow_down",
				"interval": 1,
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "token",
			"token_type":   "bearer",
		})
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	code := DeviceCode{DeviceCode: "device", ExpiresIn: 5, Interval: 1}
	token, err := pollDeviceToken(ctx, server.Client(), server.URL, "client-id", code)
	if err != nil {
		t.Fatalf("pollDeviceToken returned error: %v", err)
	}
	if token.AccessToken != "token" {
		t.Fatalf("access token = %q, want token", token.AccessToken)
	}
}
