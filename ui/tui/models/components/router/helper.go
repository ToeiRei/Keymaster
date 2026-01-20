package router

// TODO rewrite with util.Model in mind

import tea "github.com/charmbracelet/bubbletea"

func (r *Router) activeModelGet() tea.Model {
	return r.model_stack[len(r.model_stack)-1]
}

func (r *Router) activeModelSet(model tea.Model) {
	r.model_stack[len(r.model_stack)-1] = model
}

func (r *Router) activeModelPop() tea.Model {
	model := r.model_stack[len(r.model_stack)-1]
	r.model_stack = r.model_stack[:len(r.model_stack)-1]
	return model
}

func (r *Router) activeModelUpdate(msg tea.Msg) tea.Cmd {
	model, cmd := r.activeModelGet().Update(msg)
	r.activeModelSet(model)
	return cmd
}
