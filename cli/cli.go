package main

import (
	"flag"
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os"
)

var (
	baseStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	apiGateway = flag.String("api-gateway", "http://localhost", "API Gateway URL")
)

type model struct {
	warehouses table.Model
	help       help.Model
	keys       keyMap
	spinner    spinner.Model

	fetchingWarehouses bool
}

func (m model) Init() tea.Cmd {
	m.fetchingWarehouses = true
	return tea.Batch(FetchWarehouses, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.warehouses.SetHeight(msg.Height - 3)
		m.help.Width = msg.Width

	case NewWarehousesMsg:
		m.fetchingWarehouses = false
		var rows []table.Row
		for _, warehouse := range msg.warehouses {
			rows = append(rows, []string{warehouse, "--currently empty--"})
		}
		m.warehouses.SetRows(rows)

	// Is it a key press?
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Up):
			m.warehouses.MoveUp(1)
		case key.Matches(msg, m.keys.Down):
			m.warehouses.MoveDown(1)
		case key.Matches(msg, m.keys.PageUp):
			m.warehouses.GotoTop()
		case key.Matches(msg, m.keys.PageDown):
			m.warehouses.GotoBottom()
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Quit     key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m model) View() string {
	out := baseStyle.Render(m.warehouses.View())
	out += "\n"
	if m.fetchingWarehouses {
		out += m.spinner.View() + "fetching warehouses "
	}
	out += m.help.View(m.keys)
	return out
}

func main() {
	flag.Parse()
	fmt.Println(*apiGateway)

	columns := []table.Column{
		{Title: "ID", Width: 4},
		{Title: "Metadata", Width: 19},
	}

	tab := table.New(table.WithColumns(columns))
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	tab.SetStyles(s)

	h := help.New()

	keys := keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pageup", "g"),
			key.WithHelp("PgUp/g", "go to top"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pagedown", "G"),
			key.WithHelp("PgDown/G", "go to bottom"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}

	spin := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
	)

	p := tea.NewProgram(model{tab, h, keys, spin, true})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
