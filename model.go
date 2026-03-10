package main

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// borderPadding is the horizontal space consumed by each pane's rounded
	// border and internal padding. Subtracted from half the terminal width
	// to get the usable content width per pane.
	borderPadding = 3

	// chromeHeight is the vertical space consumed by the title bar, status
	// bar, and border chrome.
	chromeHeight = 5

	// filterHeight is the vertical space consumed by the filter input
	// inside the left pane.
	filterHeight = 1

	// paneCount is the number of panes for tab cycling.
	paneCount = 2
)

type viewMode int

const (
	modeJSON viewMode = iota
	modeImage
)

type model struct {
	data         map[string]any
	rawJSON      []byte
	filename     string
	jsonContent  string
	hasImage     bool
	img          image.Image
	hqImageCache string
	filterInput  textinput.Model
	jsonPane     viewport.Model
	imagePane    viewport.Model
	width        int
	height       int
	mode         viewMode
	activePane   int
	ready        bool

	// imageFields tracks JSON keys whose values are base64-encoded image
	// data. Populated during load so that formatValue can display a
	// placeholder without re-decoding.
	imageFields map[string]bool
}

func initialModel(filename string) (model, error) {
	ti := textinput.New()
	ti.Prompt = "Filter: "
	ti.Placeholder = "gjson path (e.g. users.#.name)"
	// Unbind keys that conflict with pane switching and viewport scrolling.
	ti.KeyMap.AcceptSuggestion = key.NewBinding(key.WithDisabled())
	ti.KeyMap.NextSuggestion = key.NewBinding(key.WithDisabled())
	ti.KeyMap.PrevSuggestion = key.NewBinding(key.WithDisabled())

	styles := ti.Styles()
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	ti.SetStyles(styles)
	ti.Focus()

	m := model{
		filename:    filename,
		filterInput: ti,
		imageFields: make(map[string]bool),
	}

	raw, err := os.ReadFile(filename)
	if err != nil {
		return m, fmt.Errorf("reading file: %w", err)
	}
	m.rawJSON = raw

	if err := json.Unmarshal(raw, &m.data); err != nil {
		return m, fmt.Errorf("parsing JSON: %w", err)
	}

	m.jsonContent = formatJSON(m.data, m.imageFields)

	if imgData := findImageData(m.data, m.imageFields); imgData != "" {
		img, err := decodeImage(imgData)
		if err != nil {
			log.Printf("warning: found image data but failed to decode: %v", err)
		} else {
			m.img = img
			m.hasImage = true
		}
	}

	return m, nil
}

func (m model) Init() tea.Cmd {
	return m.filterInput.Focus()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		halfWidth := m.width/2 - borderPadding
		contentHeight := m.height - chromeHeight
		jsonHeight := contentHeight - filterHeight

		m.filterInput.SetWidth(halfWidth)

		if !m.ready {
			m.jsonPane = viewport.New()
			m.jsonPane.SetWidth(halfWidth)
			m.jsonPane.SetHeight(jsonHeight)
			m.jsonPane.SetContent(m.jsonContent)

			m.imagePane = viewport.New()
			m.imagePane.SetWidth(halfWidth)
			m.imagePane.SetHeight(contentHeight)
			if m.hasImage {
				m.imagePane.SetContent(renderPreview(m.img, halfWidth-2, contentHeight))
			} else {
				m.imagePane.SetContent("\n  No image data found")
			}

			m.ready = true
		} else {
			m.jsonPane.SetWidth(halfWidth)
			m.jsonPane.SetHeight(jsonHeight)
			m.imagePane.SetWidth(halfWidth)
			m.imagePane.SetHeight(contentHeight)
			if m.hasImage {
				m.imagePane.SetContent(renderPreview(m.img, halfWidth-2, contentHeight))
			}
		}

		if m.mode == modeImage && m.hasImage {
			m.hqImageCache = renderFullscreen(m.img, m.width, m.height)
		}
	}

	// Forward non-key messages to the text input for cursor blink.
	if m.mode == modeJSON {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "esc":
		if m.mode == modeImage {
			m.mode = modeJSON
			return m, func() tea.Msg { return tea.ClearScreen() }
		}
		if m.filterInput.Value() != "" {
			m.filterInput.SetValue("")
			m.jsonContent = applyFilter(m.rawJSON, "", m.data, m.imageFields)
			if m.ready {
				m.jsonPane.SetContent(m.jsonContent)
				m.jsonPane.GotoTop()
			}
			return m, nil
		}
		return m, tea.Quit

	case "tab":
		if m.mode == modeJSON {
			m.activePane = (m.activePane + 1) % paneCount
			return m, nil
		}

	case "ctrl+p":
		if m.mode == modeJSON && m.hasImage {
			m.mode = modeImage
			m.hqImageCache = renderFullscreen(m.img, m.width, m.height)
			return m, nil
		}

	case "up", "down", "pgup", "pgdown":
		if m.ready && m.mode == modeJSON {
			var cmd tea.Cmd
			if m.activePane == 0 {
				m.jsonPane, cmd = m.jsonPane.Update(msg)
			} else {
				m.imagePane, cmd = m.imagePane.Update(msg)
			}
			return m, cmd
		}

	default:
		if m.mode == modeJSON {
			prev := m.filterInput.Value()
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			if m.filterInput.Value() != prev {
				m.jsonContent = applyFilter(m.rawJSON, m.filterInput.Value(), m.data, m.imageFields)
				if m.ready {
					m.jsonPane.SetContent(m.jsonContent)
					m.jsonPane.GotoTop()
				}
			}
			return m, cmd
		}
	}

	return m, nil
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Padding(0, 1)

	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	inactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true)
)

func (m model) View() tea.View {
	if m.mode == modeImage {
		v := tea.NewView("Press esc to return\n\n" + m.hqImageCache)
		v.AltScreen = true
		return v
	}

	if !m.ready {
		return tea.NewView("Loading...")
	}

	title := titleStyle.Render(fmt.Sprintf("Glimpse — %s", m.filename))

	halfWidth := m.width/2 - borderPadding
	contentHeight := m.height - chromeHeight

	leftStyle := inactiveBorder.Width(halfWidth).Height(contentHeight)
	rightStyle := inactiveBorder.Width(halfWidth).Height(contentHeight)
	if m.activePane == 0 {
		leftStyle = activeBorder.Width(halfWidth).Height(contentHeight)
	} else {
		rightStyle = activeBorder.Width(halfWidth).Height(contentHeight)
	}

	filterView := strings.TrimRight(m.filterInput.View(), "\n")
	leftInner := filterView + "\n" + m.jsonPane.View()
	left := leftStyle.Render(leftInner)
	right := rightStyle.Render(m.imagePane.View())

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	imgHint := ""
	if m.hasImage {
		imgHint = accentStyle.Render(" • ctrl+p: HD image")
	}
	status := statusStyle.Render(
		fmt.Sprintf("  esc: clear/quit • tab: switch pane • ↑↓: scroll%s", imgHint))

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, title, content, status))
	v.AltScreen = true
	return v
}
