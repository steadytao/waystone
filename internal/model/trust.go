// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type TrustPolicy struct {
	Version           string            `json:"version"`
	UpdatedAt         time.Time         `json:"updated_at"`
	TrustedIdentities []TrustedIdentity `json:"trusted_identities,omitempty"`
}

type TrustedIdentity struct {
	ID        string    `json:"id"`
	TrustedAt time.Time `json:"trusted_at"`
}

func (p TrustPolicy) Trusts(identityID string) bool {
	for _, identity := range p.TrustedIdentities {
		if identity.ID == identityID {
			return true
		}
	}
	return false
}
