// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"github.com/toeirei/keymaster/internal/keys"
	"github.com/toeirei/keymaster/internal/model"
)

// DeploymentPlan is a pure description of what should be deployed to a host.
// It contains the account to create, the keys to assign, and the final
// authorized_keys content. Building this plan must not perform any side-effects.
type DeploymentPlan struct {
	Account            model.Account
	SelectedKeyIDs     []int
	GlobalKeys         []model.PublicKey
	AccountKeys        []model.PublicKey
	AuthorizedKeysBlob string
}

// BuildBootstrapDeploymentPlan constructs a DeploymentPlan from the provided
// parameters and key material. It is pure/deterministic: callers must fetch
// keys and system key beforehand and pass them in.
func BuildBootstrapDeploymentPlan(params BootstrapParams, systemKey *model.SystemKey, globalKeys, accountKeys []model.PublicKey) (DeploymentPlan, error) {
	plan := DeploymentPlan{}

	plan.Account = model.Account{
		Username: params.Username,
		Hostname: params.Hostname,
		Label:    params.Label,
		Tags:     params.Tags,
		IsActive: true,
	}

	plan.SelectedKeyIDs = append([]int{}, params.SelectedKeyIDs...)
	plan.GlobalKeys = append([]model.PublicKey{}, globalKeys...)
	plan.AccountKeys = append([]model.PublicKey{}, accountKeys...)

	// Build authorized_keys content using the pure keys helper.
	content, err := keys.BuildAuthorizedKeysContent(systemKey, globalKeys, accountKeys)
	if err != nil {
		return DeploymentPlan{}, err
	}
	plan.AuthorizedKeysBlob = content

	return plan, nil
}

