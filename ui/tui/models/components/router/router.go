package router

// TODO rewrite with util.Model in mind

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/toeirei/keymaster/ui/tui/util"
)

var routerId = 1

type Router struct {
	id          int
	size        util.Size
	model_stack []tea.Model
}

var _ tea.Model = (*Router)(nil)

func New(initial_model tea.Model) (Router, RouterControll) {
	routerId++
	return Router{
			id:          routerId - 1,
			model_stack: []tea.Model{initial_model},
		}, RouterControll{
			rid: routerId - 1,
		}
}

func (r Router) Init() tea.Cmd {
	return tea.Batch(
		r.activeModelGet().Init(),
		r.activeModelUpdate(InitMsg{RouterControll: RouterControll{rid: r.id}}),
	)
}

func (r Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if r.size.Update(msg) {
		// pass window size update
		cmds = append(cmds, r.activeModelUpdate(msg))
	} else if r.isMsgOwner(msg) {
		// handle controll messages meant for this router
		switch msg := msg.(type) {
		case PushMsg:
			cmds = append(cmds, r.handlePush(msg)...)
		case PopMsg:
			cmds = append(cmds, r.handlePop(msg)...)
		case ChangeMsg:
			cmds = append(cmds, r.handleChange(msg)...)
		}
	} else if IsRouterMsg(msg) {
		// rewrite info messages from other routers
		switch msg := msg.(type) {
		case InitMsg:
			// do not pass init messages, to prevent childs from obtaining parent routers RouterControll
		case SuspendMsg:
			cmds = append(cmds, r.handleSuspend(msg))
		case ResumeMsg:
			cmds = append(cmds, r.handleResume(msg))
		case DestroyMsg:
			cmds = append(cmds, r.handleDestroy(msg))
		default:
			// pass controll messages for child routers
			cmds = append(cmds, r.activeModelUpdate(msg))
		}
	} else {
		// pass other update
		cmds = append(cmds, r.activeModelUpdate(msg))
	}

	return r, tea.Batch(cmds...)
}

func (r Router) View() string {
	return r.activeModelGet().View()
}

// handle PushMsg
func (r *Router) handlePush(msg PushMsg) []tea.Cmd {
	var cmds []tea.Cmd
	// suspend recent model
	cmds = append(cmds, r.activeModelUpdate(SuspendMsg{rid: r.id}))
	// push new model
	r.model_stack = append(r.model_stack, msg.Model)
	// initialize pushed model
	cmds = append(cmds, msg.Model.Init())
	cmds = append(cmds, r.activeModelUpdate(InitMsg{RouterControll: RouterControll{rid: r.id}}))
	return append(cmds, r.activeModelUpdate(tea.WindowSizeMsg(r.size)))
}

// handle PopMsg
func (r *Router) handlePop(msg PopMsg) []tea.Cmd {
	var cmds []tea.Cmd
	// pop and destroy old models
	for range msg.Count {
		if len(r.model_stack) <= 1 {
			break
		}
		_, cmd := r.activeModelPop().Update(DestroyMsg{rid: r.id})
		cmds = append(cmds, cmd)
	}
	// resume active model
	return append(cmds, r.activeModelUpdate(ResumeMsg{rid: r.id}))
}

// handle ChangeMsg
func (r *Router) handleChange(msg ChangeMsg) []tea.Cmd {
	var cmds []tea.Cmd
	// destroy recent model
	cmds = append(cmds, r.activeModelUpdate(DestroyMsg{rid: r.id}))
	// set new model
	r.activeModelSet(msg.Model)
	// initialize set model
	cmds = append(cmds, msg.Model.Init())
	cmds = append(cmds, r.activeModelUpdate(InitMsg{RouterControll: RouterControll{rid: r.id}}))
	return append(cmds, r.activeModelUpdate(tea.WindowSizeMsg(r.size)))
}

// handle SuspendMsg (from other router)
func (r *Router) handleSuspend(_ SuspendMsg) tea.Cmd {
	return r.activeModelUpdate(SuspendMsg{rid: r.id})
}

// handle ResumeMsg (from other router)
func (r *Router) handleResume(_ ResumeMsg) tea.Cmd {
	return r.activeModelUpdate(ResumeMsg{rid: r.id})
}

// handle DestroyMsg (from other router)
func (r *Router) handleDestroy(_ DestroyMsg) tea.Cmd {
	return r.activeModelUpdate(DestroyMsg{rid: r.id})
}

func (r *Router) isMsgOwner(msg tea.Msg) bool {
	rmsg, ok := msg.(RouterMsg)
	return ok && rmsg.routerId() == r.id
}
