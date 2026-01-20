// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/util/slicest"
)

func UpdateTeaModelInplace[M any](msg tea.Msg, model *M) tea.Cmd {
	var model_any any = *model

	if model_updatable, ok := model_any.(updatableSelf[M]); ok {
		model_updated, cmd := model_updatable.Update(msg)
		*model = model_updated
		return cmd
	}

	if model_updatable, ok := model_any.(updatableTea); ok {
		model_updated, cmd := model_updatable.Update(msg)
		if model_updated, ok := model_updated.(M); ok {
			*model = model_updated
			return cmd
		}
		return cmd
	}

	// no supported update method
	return nil
}

func UpdateTeaModelsInplace[M any](msg tea.Msg, models ...*M) tea.Cmd {
	return tea.Batch(slicest.Map(models, func(model *M) tea.Cmd {
		return UpdateTeaModelInplace(msg, model)
	})...)
}

type updatableTea interface {
	Update(tea.Msg) (Model, tea.Cmd)
}
type updatableSelf[T any] interface {
	Update(tea.Msg) (T, tea.Cmd)
}
