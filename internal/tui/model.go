package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/d-init-d/d-research-cli/internal/app"
	"github.com/d-init-d/d-research-cli/internal/doctor"
	"github.com/d-init-d/d-research-cli/internal/event"
	"github.com/d-init-d/d-research-cli/internal/graph"
	"github.com/d-init-d/d-research-cli/internal/host"
	"github.com/d-init-d/d-research-cli/internal/i18n"
)

type viewMode int

const (
	viewStream viewMode = iota
	viewGraph
	viewSplit
)

type rightTab int

const (
	tabPlan rightTab = iota
	tabEvidence
	tabGaps
	tabBlockers
	tabArtifacts
)

type screen int

const (
	screenMain screen = iota
	screenSettings
	screenPalette
	screenConnect
)

type Model struct {
	svc          *app.Service
	width        int
	height       int
	view         viewMode
	tab          rightTab
	screen       screen
	settingsTab  int
	events       []event.Event
	streamVP     viewport.Model
	graphVP      viewport.Model
	rightVP      viewport.Model
	input        textinput.Model
	prompt       string
	bus          *event.Bus
	evCh         chan event.Event
	awaitApprove bool
	selectedNode string
	paletteItems []paletteItem
	paletteIdx   int
}

type paletteItem struct {
	Name        string
	Description string
	Action      func(Model) tea.Cmd
}

func NewModel(svc *app.Service) Model {
	ti := textinput.New()
	ti.Placeholder = "Nhập câu hỏi nghiên cứu hoặc lệnh (/settings, /connect)"
	ti.CharLimit = 500
	ti.Width = 60
	ti.Focus()
	stream := viewport.New(40, 20)
	graph := viewport.New(40, 20)
	right := viewport.New(30, 20)
	bus := svc.EventBus()
	evCh := make(chan event.Event, 64)
	bus.Subscribe(func(ev event.Event) {
		select {
		case evCh <- ev:
		default:
		}
	})
	return Model{
		svc:          svc,
		streamVP:     stream,
		graphVP:      graph,
		rightVP:      right,
		input:        ti,
		bus:          bus,
		evCh:         evCh,
		paletteItems: defaultPalette(),
	}
}

func Run(svc *app.Service) error {
	m := NewModel(svc)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func waitForBusEvent(ch <-chan event.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return busEventMsg{event: ev}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, waitForBusEvent(m.evCh))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layoutViewports()
		return m, nil
	case busEventMsg:
		m.appendEvent(msg.event)
		if msg.event.Kind == "approval_required" {
			m.awaitApprove = true
		}
		m.refreshStream()
		m.refreshRight()
		return m, waitForBusEvent(m.evCh)
	case runFinishedMsg:
		if msg.awaitApprove {
			m.awaitApprove = true
		}
		if msg.err != nil {
			m.bus.PublishSimple("research", "system", "error", "failed", msg.err.Error(), "")
		}
		m.refreshStream()
		m.refreshRight()
		return m, nil
	case approveFinishedMsg:
		m.awaitApprove = false
		if msg.err != nil && !strings.Contains(msg.err.Error(), "approval") {
			m.bus.PublishSimple("research", "system", "error", "failed", msg.err.Error(), "")
		}
		m.refreshStream()
		m.refreshRight()
		return m, nil
	case doctorFinishedMsg:
		m.bus.PublishSimple("research", "system", "doctor", "done", fmt.Sprintf("doctor status=%s ok=%v %s", msg.status, msg.ok, msg.detail), "")
		m.refreshStream()
		return m, nil
	case screenChangeMsg:
		m.screen = screen(msg)
		if m.screen == screenPalette {
			m.paletteIdx = 0
		}
		return m, nil
	case tea.KeyMsg:
		if m.screen == screenPalette {
			return m.updatePalette(msg)
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.screen != screenMain {
				m.screen = screenMain
				return m, nil
			}
			return m, tea.Quit
		case "tab":
			m.tab = (m.tab + 1) % 5
			m.refreshRight()
			return m, nil
		case "1":
			m.view = viewStream
			return m, nil
		case "2":
			m.view = viewGraph
			m.refreshGraph()
			return m, nil
		case "3":
			m.view = viewSplit
			m.refreshGraph()
			return m, nil
		case "enter":
			if m.awaitApprove {
				return m, m.approvePlan()
			}
			text := strings.TrimSpace(m.input.Value())
			if strings.HasPrefix(text, "/") {
				return m, m.handleCommand(text)
			}
			if text != "" {
				m.prompt = text
				m.input.SetValue("")
				return m, m.startRun(text)
			}
		case "a":
			if m.awaitApprove {
				return m, m.approvePlan()
			}
		}
	}
	var cmd tea.Cmd
	if m.screen == screenMain || m.screen == screenSettings || m.screen == screenConnect {
		m.input, cmd = m.input.Update(msg)
	}
	return m, cmd
}

func (m *Model) appendEvent(ev event.Event) {
	m.events = append(m.events, ev)
	if len(m.events) > 200 {
		m.events = m.events[len(m.events)-200:]
	}
}

func (m *Model) layoutViewports() {
	colW := (m.width - 4) / 3
	if colW < 20 {
		colW = 20
	}
	h := m.height - 4
	if h < 8 {
		h = 8
	}
	m.streamVP.Width = colW
	m.streamVP.Height = h
	m.graphVP.Width = colW
	m.graphVP.Height = h
	m.rightVP.Width = colW
	m.rightVP.Height = h
	m.input.Width = m.width - 4
	m.refreshStream()
	m.refreshGraph()
	m.refreshRight()
}

func (m Model) View() string {
	switch m.screen {
	case screenSettings:
		return m.renderSettings()
	case screenPalette:
		return m.renderPalette()
	case screenConnect:
		return m.renderConnect()
	default:
		return m.renderMain()
	}
}

func (m Model) renderMain() string {
	left := m.renderLeft()
	center := m.renderCenter()
	right := m.renderRightPanel()
	cols := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"[1] Dòng  [2] Graph  [3] Chia đôi  |  Tab: đổi panel phải  |  /settings /connect /palette",
	)
	if m.awaitApprove {
		footer += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Plan chờ phê duyệt — nhấn Enter hoặc `a` để duyệt")
	}
	return lipgloss.JoinVertical(lipgloss.Left, cols, m.input.View(), footer)
}

func (m Model) renderLeft() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("app.title")) + "\n")
	b.WriteString(subtleStyle.Render(i18n.T("app.subtitle")) + "\n\n")
	b.WriteString(sectionStyle.Render(i18n.T("nav.overview")) + "\n")
	if m.prompt != "" {
		b.WriteString(fmt.Sprintf("Prompt: %s\n", truncate(m.prompt, 40)))
	} else {
		b.WriteString("Chưa có phiên chạy\n")
	}
	b.WriteString("\n" + sectionStyle.Render(i18n.T("nav.agents")) + "\n")
	agent := latestAgent(m.events)
	if agent == "" {
		b.WriteString("—\n")
	} else {
		b.WriteString(agent + "\n")
	}
	b.WriteString("\n" + sectionStyle.Render(i18n.T("nav.usage")) + "\n")
	b.WriteString(fmt.Sprintf("Events: %d\n", len(m.events)))
	return panelStyle.Width((m.width - 4) / 3).Render(b.String())
}

func (m Model) renderCenter() string {
	switch m.view {
	case viewGraph:
		return panelStyle.Width((m.width - 4) / 3).Render(m.graphVP.View())
	case viewSplit:
		top := m.streamVP.View()
		bottom := m.graphVP.View()
		body := lipgloss.JoinVertical(lipgloss.Left, top, bottom)
		return panelStyle.Width((m.width - 4) / 3).Render(body)
	default:
		return panelStyle.Width((m.width - 4) / 3).Render(m.streamVP.View())
	}
}

func (m Model) renderRightPanel() string {
	return panelStyle.Width((m.width - 4) / 3).Render(m.rightVP.View())
}

func (m *Model) refreshStream() {
	var lines []string
	for _, ev := range m.events {
		line := fmt.Sprintf("[%s] %s %s: %s", ev.Time.Format("15:04:05"), ev.Agent, ev.Kind, ev.Message)
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{"(chưa có event)"}
	}
	m.streamVP.SetContent(strings.Join(lines, "\n"))
	m.streamVP.GotoBottom()
}

func (m *Model) refreshGraph() {
	proj, err := graph.Project(m.svc.CWD, 42, graph.DefaultMaxNodes, m.selectedNode, 2)
	text := graph.RenderASCII(proj, m.selectedNode)
	if err != nil {
		text = err.Error()
	}
	m.graphVP.SetContent(text)
}

func (m *Model) refreshRight() {
	tabs := []string{i18n.T("tab.plan"), i18n.T("tab.evidence"), i18n.T("tab.gaps"), i18n.T("tab.blockers"), i18n.T("tab.artifacts")}
	var b strings.Builder
	for i, t := range tabs {
		if int(m.tab) == i {
			b.WriteString("[" + t + "] ")
		} else {
			b.WriteString(t + " ")
		}
	}
	b.WriteString("\n\n")
	switch m.tab {
	case tabPlan:
		b.WriteString("research-plan.json\nTrạng thái: ")
		if m.awaitApprove {
			b.WriteString(i18n.T("status.awaiting"))
		} else if m.prompt != "" {
			b.WriteString(i18n.T("status.running"))
		} else {
			b.WriteString("—")
		}
	case tabEvidence:
		b.WriteString("Evidence ledger sẽ cập nhật theo event stream.")
	case tabGaps:
		b.WriteString("Gaps chưa được tổng hợp.")
	case tabBlockers:
		b.WriteString("Blockers chưa được tổng hợp.")
	default:
		b.WriteString("Artifacts: research-output/")
	}
	m.rightVP.SetContent(b.String())
}

func (m Model) startRun(prompt string) tea.Cmd {
	return func() tea.Msg {
		ws, err := m.svc.Workspace()
		if err != nil {
			return runFinishedMsg{err: err}
		}
		h := host.New(ws, m.bus, host.Options{Mode: "research"})
		err = h.Run(context.Background(), prompt)
		if err == nil {
			st, loadErr := ws.LoadState()
			if loadErr == nil && st.Phase == "awaiting_approval" {
				return runFinishedMsg{awaitApprove: true}
			}
			return runFinishedMsg{}
		}
		if strings.Contains(err.Error(), "approval") {
			return runFinishedMsg{awaitApprove: true}
		}
		return runFinishedMsg{err: err}
	}
}

func (m Model) approvePlan() tea.Cmd {
	return func() tea.Msg {
		ws, err := m.svc.Workspace()
		if err != nil {
			return approveFinishedMsg{err: err}
		}
		h := host.New(ws, m.bus, host.Options{Mode: "research", Resume: true})
		if err := h.ApprovePlan("tui-user", "approved via TUI"); err != nil {
			return approveFinishedMsg{err: err}
		}
		err = h.Run(context.Background(), m.prompt)
		return approveFinishedMsg{err: err}
	}
}

func (m Model) handleCommand(text string) tea.Cmd {
	switch strings.Fields(text)[0] {
	case "/settings":
		return func() tea.Msg { return screenChangeMsg(screenSettings) }
	case "/connect":
		return func() tea.Msg { return screenChangeMsg(screenConnect) }
	case "/palette":
		return func() tea.Msg { return screenChangeMsg(screenPalette) }
	case "/search":
		return func() tea.Msg {
			return screenChangeMsg(screenSettings)
		}
	case "/browser":
		return func() tea.Msg {
			return screenChangeMsg(screenSettings)
		}
	default:
		m.bus.PublishSimple("research", "system", "command", "unknown", "Lệnh không rõ: "+text, "")
	}
	return nil
}

func (m Model) renderSettings() string {
	sections := []string{i18n.T("settings.models"), i18n.T("settings.search"), i18n.T("settings.browser"), i18n.T("settings.security"), i18n.T("settings.runtime")}
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("settings.title")) + "\n\n")
	for i, s := range sections {
		if i == m.settingsTab {
			b.WriteString("> " + s + "\n")
		} else {
			b.WriteString("  " + s + "\n")
		}
	}
	b.WriteString("\nDùng CLI tương ứng: model/search/browser/auth\nEsc để quay lại")
	return panelStyle.Width(m.width - 2).Render(b.String()) + "\n" + m.input.View()
}

func (m Model) renderConnect() string {
	body := "Wizard kết nối provider/model\nChạy: d-research auth login --provider openrouter\nEsc để quay lại"
	return panelStyle.Width(m.width - 2).Render(body) + "\n" + m.input.View()
}

func (m Model) renderPalette() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(i18n.T("cmd.palette")) + "\n\n")
	for i, item := range m.paletteItems {
		prefix := "  "
		if i == m.paletteIdx {
			prefix = "> "
		}
		b.WriteString(prefix + item.Name + " — " + item.Description + "\n")
	}
	b.WriteString("\n↑/↓ chọn, Enter thực thi, Esc đóng")
	return panelStyle.Width(m.width - 2).Render(b.String())
}

func (m Model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.paletteIdx > 0 {
			m.paletteIdx--
		}
	case "down", "j":
		if m.paletteIdx < len(m.paletteItems)-1 {
			m.paletteIdx++
		}
	case "enter":
		item := m.paletteItems[m.paletteIdx]
		m.screen = screenMain
		if item.Action != nil {
			return m, item.Action(m)
		}
	case "esc":
		m.screen = screenMain
	}
	return m, nil
}

func defaultPalette() []paletteItem {
	return []paletteItem{
		{Name: "/settings", Description: "Mở cài đặt", Action: func(m Model) tea.Cmd {
			return func() tea.Msg {
				return screenChangeMsg(screenSettings)
			}
		}},
		{Name: "/connect", Description: "Wizard provider", Action: func(m Model) tea.Cmd {
			return func() tea.Msg {
				return screenChangeMsg(screenConnect)
			}
		}},
		{Name: "doctor", Description: "Chạy doctor", Action: func(m Model) tea.Cmd {
			return func() tea.Msg {
				rep, err := m.svc.Doctor()
				if err != nil {
					return doctorFinishedMsg{ok: false, status: doctor.StatusFailed, detail: err.Error()}
				}
				return doctorFinishedMsg{ok: rep.OK, status: rep.Status, detail: fmt.Sprintf("checks=%d", len(rep.Checks))}
			}
		}},
	}
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	subtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	panelStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

func latestAgent(events []event.Event) string {
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Agent != "" && events[i].Agent != "system" {
			return events[i].Agent
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// KeyMap for future help overlay.
type KeyMap struct {
	Quit key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding { return []key.Binding{k.Quit} }
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit}}
}