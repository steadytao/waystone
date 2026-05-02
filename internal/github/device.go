// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/steadytao/waystone/internal/auth"
)

const deviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"

type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

func RequestDeviceCode(ctx context.Context, client *http.Client, clientID, scope string) (DeviceCode, error) {
	return requestDeviceCode(ctx, client, "https://github.com/login/device/code", clientID, scope)
}

func requestDeviceCode(ctx context.Context, client *http.Client, endpoint, clientID, scope string) (DeviceCode, error) {
	values := url.Values{"client_id": {clientID}}
	if scope != "" {
		values.Set("scope", scope)
	}

	var code DeviceCode
	if err := postOAuth(ctx, client, endpoint, values, &code); err != nil {
		return DeviceCode{}, err
	}
	if code.Interval == 0 {
		code.Interval = 5
	}
	return code, nil
}

func PollDeviceToken(ctx context.Context, client *http.Client, clientID string, code DeviceCode) (auth.GitHubToken, error) {
	return pollDeviceToken(ctx, client, "https://github.com/login/oauth/access_token", clientID, code)
}

func pollDeviceToken(ctx context.Context, client *http.Client, endpoint, clientID string, code DeviceCode) (auth.GitHubToken, error) {
	if code.DeviceCode == "" {
		return auth.GitHubToken{}, errors.New("device code must not be empty")
	}

	interval := time.Duration(code.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	expiresIn := time.Duration(code.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = 15 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, expiresIn)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return auth.GitHubToken{}, ctx.Err()
		case <-time.After(interval):
		}

		values := url.Values{
			"client_id":   {clientID},
			"device_code": {code.DeviceCode},
			"grant_type":  {deviceGrantType},
		}

		var response struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			Scope       string `json:"scope"`
			Error       string `json:"error"`
			Description string `json:"error_description"`
			Interval    int    `json:"interval"`
		}
		if err := postOAuth(ctx, client, endpoint, values, &response); err != nil {
			return auth.GitHubToken{}, err
		}

		switch response.Error {
		case "":
			if response.AccessToken == "" {
				return auth.GitHubToken{}, errors.New("github returned an empty access token")
			}
			return auth.GitHubToken{
				AccessToken: response.AccessToken,
				TokenType:   response.TokenType,
				Scope:       response.Scope,
				CreatedAt:   time.Now().UTC(),
			}, nil
		case "authorization_pending":
			continue
		case "slow_down":
			if response.Interval > 0 {
				interval = time.Duration(response.Interval) * time.Second
			} else {
				interval += 5 * time.Second
			}
			continue
		default:
			if response.Description != "" {
				return auth.GitHubToken{}, fmt.Errorf("github device flow failed: %s: %s", response.Error, response.Description)
			}
			return auth.GitHubToken{}, fmt.Errorf("github device flow failed: %s", response.Error)
		}
	}
}

func postOAuth(ctx context.Context, client *http.Client, endpoint string, values url.Values, out any) error {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "waystone")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github oauth request failed: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
