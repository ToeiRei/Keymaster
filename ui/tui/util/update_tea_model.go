// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/util/slicest"
)

func UpdateTeaModelInplace[M any](msg tea.Msg, model *M) tea.Cmd {
	var modelAny any = *model

	if modelUpdatable, ok := modelAny.(updatableSelf[M]); ok {
		modelUpdated, cmd := modelUpdatable.Update(msg)
		*model = modelUpdated
		return cmd
	}

	if modelUpdatable, ok := modelAny.(updatableTea); ok {
		modelUpdated, cmd := modelUpdatable.Update(msg)
		if modelUpdated, ok := modelUpdated.(M); ok {
			*model = modelUpdated
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
