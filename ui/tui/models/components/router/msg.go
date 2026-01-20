package router

// TODO rewrite with util.Model in mind

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Router invoked messages
type InitMsg struct {
	RouterControll RouterControll
}
type SuspendMsg struct {
	rid int
}
type ResumeMsg struct {
	rid int
}
type DestroyMsg struct {
	rid int
}

// RouterControll invoked messages
type PushMsg struct {
	rid   int
	Model tea.Model
}
type PopMsg struct {
	rid   int
	Count int
}
type ChangeMsg struct {
	rid   int
	Model tea.Model
}

func (m InitMsg) routerId() int    { return m.RouterControll.rid }
func (m SuspendMsg) routerId() int { return m.rid }
func (m ResumeMsg) routerId() int  { return m.rid }
func (m DestroyMsg) routerId() int { return m.rid }
func (m PushMsg) routerId() int    { return m.rid }
func (m PopMsg) routerId() int     { return m.rid }
func (m ChangeMsg) routerId() int  { return m.rid }

type RouterMsg interface {
	routerId() int
}

func IsRouterMsg(msg tea.Msg) bool {
	_, ok := msg.(RouterMsg)
	return ok
}
