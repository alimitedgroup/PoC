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
	"strconv"
)

var (
	baseStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	apiGateway = flag.String("api-gateway", "http://localhost", "API Gateway URL")
)

type model struct {
	warehouses table.Model
	stock      table.Model
	help       help.Model
	spinner    spinner.Model
	keys       keyMap
	// selectedWarehouse is empty if no warehouse is selected (and we're
	// browsing all of them), otherwise, if the stock of a warehouse is
	// currently shown, it contains the id of that warehouse
	selectedWarehouse  string
	fetchingWarehouses bool
	fetchingStock      bool
}

func (m model) Init() tea.Cmd {
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
		m.stock.SetHeight(msg.Height - 3)
		m.help.Width = msg.Width

	case NewWarehousesMsg:
		m.fetchingWarehouses = false
		m.keys.Refresh.SetEnabled(true)

		var rows []table.Row
		for _, warehouse := range msg.warehouses {
			rows = append(rows, []string{warehouse, "--currently empty--"})
		}
		m.warehouses.SetRows(rows)

	case NewStockMsg:
		m.fetchingStock = false
		m.keys.Refresh.SetEnabled(true)

		var rows []table.Row
		for good, amount := range msg.stock {
			rows = append(rows, []string{good, strconv.Itoa(amount)})
		}
		m.stock.SetRows(rows)

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

		case key.Matches(msg, m.keys.Refresh):
			m.keys.Refresh.SetEnabled(false)
			if m.selectedWarehouse == "" {
				m.fetchingWarehouses = true
				return m, FetchWarehouses
			} else {
				m.fetchingStock = true
				return m, FetchStock(m.selectedWarehouse)
			}
		case key.Matches(msg, m.keys.Select):
			m.fetchingStock = true
			m.keys.Select.SetEnabled(false)
			m.keys.Refresh.SetEnabled(false)
			m.selectedWarehouse = m.warehouses.SelectedRow()[0]
			return m, FetchStock(m.selectedWarehouse)
		}
	}

	if m.selectedWarehouse == "" && m.warehouses.SelectedRow() != nil {
		m.keys.Select.SetEnabled(true)
	}

	return m, nil
}

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Refresh  key.Binding
	Select   key.Binding
	Quit     key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Select, k.Up, k.Down, k.PageUp, k.PageDown, k.Refresh, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}

func (m model) View() string {
	out := ""

	if m.selectedWarehouse != "" {
		out = baseStyle.Render(m.stock.View()) + "\n"
	} else {
		out = baseStyle.Render(m.warehouses.View()) + "\n"
	}

	if m.fetchingWarehouses {
		out += m.spinner.View() + "fetching warehouses "
	}
	if m.fetchingStock {
		out += m.spinner.View() + "fetching stock "
	}

	out += m.help.View(m.keys)
	return out
}

func main() {
	flag.Parse()

	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	warehouses := table.New(table.WithColumns([]table.Column{
		{Title: "ID", Width: 4},
		{Title: "Metadata", Width: 19},
	}), table.WithStyles(tableStyle))
	stock := table.New(table.WithColumns([]table.Column{
		{Title: "ID", Width: 4},
		{Title: "Amount", Width: 19},
	}), table.WithStyles(tableStyle))

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
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("R", "refresh"),
			key.WithDisabled(),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("⏎", "view stock"),
			key.WithDisabled(),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}

	spin := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
	)

	p := tea.NewProgram(model{
		warehouses:         warehouses,
		stock:              stock,
		help:               h,
		spinner:            spin,
		keys:               keys,
		selectedWarehouse:  "",
		fetchingWarehouses: true,
		fetchingStock:      false,
	})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
