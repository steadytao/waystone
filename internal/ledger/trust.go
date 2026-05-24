// Copyright 2026 The Waystone Authors
// SPDX-License-Identifier: Apache-2.0

package ledger

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/steadytao/waystone/internal/model"
)

func (r Reader) Identities() ([]model.Identity, error) {
	identities, err := readDirJSONRooted[model.Identity](r.Root, "identities")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return identities, nil
}

func (r Reader) Identity(id string) (model.Identity, error) {
	identities, err := r.Identities()
	if err != nil {
		return model.Identity{}, err
	}
	for _, identity := range identities {
		if identity.ID == id {
			return identity, nil
		}
	}
	return model.Identity{}, fmt.Errorf("identity %q not found", id)
}

func (r Reader) TrustPolicy() (model.TrustPolicy, error) {
	var policy model.TrustPolicy
	if err := r.readJSON("trust.json", &policy); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.TrustPolicy{Version: "waystone.trust.v1"}, nil
		}
		return model.TrustPolicy{}, err
	}
	if policy.Version == "" {
		policy.Version = "waystone.trust.v1"
	}
	return policy, nil
}

func (w Writer) TrustIdentity(id string) error {
	if _, err := (Reader(w)).Identity(id); err != nil {
		return err
	}
	policy, err := (Reader(w)).TrustPolicy()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	policy.Version = "waystone.trust.v1"
	policy.UpdatedAt = now
	for i, identity := range policy.TrustedIdentities {
		if identity.ID == id {
			policy.TrustedIdentities[i].TrustedAt = now
			return writeJSONUnderRoot(w.Root, "trust.json", policy)
		}
	}
	policy.TrustedIdentities = append(policy.TrustedIdentities, model.TrustedIdentity{ID: id, TrustedAt: now})
	return writeJSONUnderRoot(w.Root, "trust.json", policy)
}

func (w Writer) UntrustIdentity(id string) error {
	policy, err := (Reader(w)).TrustPolicy()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	policy.Version = "waystone.trust.v1"
	policy.UpdatedAt = now
	trusted := policy.TrustedIdentities[:0]
	for _, identity := range policy.TrustedIdentities {
		if identity.ID != id {
			trusted = append(trusted, identity)
		}
	}
	policy.TrustedIdentities = trusted
	return writeJSONUnderRoot(w.Root, "trust.json", policy)
}
