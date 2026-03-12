package main

import (
	"encoding/json"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// borderPadding accounts for the gap needed so that two panes placed
	// side-by-side don't exceed the terminal width. Each pane's lipgloss
	// Width is border-box, so the rendered width equals the Width value.
	borderPadding = 3

	// paneFrame is the horizontal space consumed by the rounded border
	// (1 left + 1 right) and padding (1 left + 1 right) inside each pane.
	// Content (viewports, filter) must be sized to (paneWidth - paneFrame)
	// so that lipgloss does not word-wrap and inflate the line count.
	paneFrame = 4

	// chromeHeight is the vertical space consumed by the title bar, status
	// bar, and border chrome.
	chromeHeight = 5

	// filterHeight is the vertical space consumed by the filter input
	// inside the left pane.
	filterHeight = 1
)

type viewMode int

const (
	modeJSON viewMode = iota
	modeImage
)

type model struct {
	data         map[string]any
	filename     string
	jsonContent  string
	images       []imageEntry
	imageIdx     int
	hdImageCache string
	filterInput  textinput.Model
	jsonPane     viewport.Model
	imagePane    viewport.Model
	gfx          graphics
	width        int
	height       int
	mode         viewMode
	activePane   int
	ready        bool
}

func initialModel(name string, raw []byte, gfx graphics) (model, error) {
	ti := textinput.New()
	ti.Prompt = "Filter: "
	ti.Placeholder = "search keys and values..."
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
		filename:    name,
		filterInput: ti,
		gfx:         gfx,
	}

	if err := json.Unmarshal(raw, &m.data); err != nil {
		return m, fmt.Errorf("parsing JSON: %w", err)
	}

	m.jsonContent = formatJSON(m.data)
	m.images = findImages(m.data)

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

		innerWidth := m.width/2 - borderPadding - paneFrame
		// Both panes share the same viewport height: total height minus
		// chrome (title + status + borders) minus the one-line header
		// each pane uses (filter bar / image selector).
		vpHeight := m.height - chromeHeight - filterHeight

		m.filterInput.SetWidth(innerWidth)

		if !m.ready {
			m.jsonPane = viewport.New()
			m.jsonPane.SetWidth(innerWidth)
			m.jsonPane.SetHeight(vpHeight)
			m.jsonPane.SetContent(m.jsonContent)

			m.imagePane = viewport.New()
			m.imagePane.SetWidth(innerWidth)
			m.imagePane.SetHeight(vpHeight)
			m.imagePane.SetContent(m.renderImageContent(innerWidth, vpHeight))

			m.ready = true
		} else {
			m.jsonPane.SetWidth(innerWidth)
			m.jsonPane.SetHeight(vpHeight)
			m.imagePane.SetWidth(innerWidth)
			m.imagePane.SetHeight(vpHeight)
			m.imagePane.SetContent(m.renderImageContent(innerWidth, vpHeight))
		}

		if m.mode == modeImage && len(m.images) > 0 {
			m.hdImageCache = renderFullscreen(m.images[m.imageIdx].img, m.width, m.height, m.gfx)
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

// refreshImagePane re-renders the image preview to match the current
// terminal dimensions and selected image.
func (m *model) refreshImagePane() {
	innerWidth := m.width/2 - borderPadding - paneFrame
	vpHeight := m.height - chromeHeight - filterHeight
	m.imagePane.SetContent(m.renderImageContent(innerWidth, vpHeight))
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
			m.jsonContent = formatJSON(m.data)
			if m.ready {
				m.jsonPane.SetContent(m.jsonContent)
				m.jsonPane.GotoTop()
			}
			return m, nil
		}
		return m, tea.Quit

	case "tab":
		if m.mode == modeJSON {
			m.activePane = 1 - m.activePane
			return m, nil
		}

	case "ctrl+p":
		if m.mode == modeJSON && len(m.images) > 0 {
			m.mode = modeImage
			m.hdImageCache = renderFullscreen(m.images[m.imageIdx].img, m.width, m.height, m.gfx)
			return m, nil
		}

	case "up", "down", "pgup", "pgdown":
		if m.ready && m.mode == modeJSON {
			if m.activePane == 0 {
				var cmd tea.Cmd
				m.jsonPane, cmd = m.jsonPane.Update(msg)
				return m, cmd
			}
			// Image pane: cycle through images.
			if len(m.images) > 1 {
				switch msg.String() {
				case "up", "pgup":
					m.imageIdx = (m.imageIdx - 1 + len(m.images)) % len(m.images)
				default:
					m.imageIdx = (m.imageIdx + 1) % len(m.images)
				}
				m.hdImageCache = ""
				m.refreshImagePane()
			}
			return m, nil
		}

	default:
		if m.mode == modeJSON {
			prev := m.filterInput.Value()
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			if m.filterInput.Value() != prev {
				m.jsonContent = applyFilter(m.filterInput.Value(), m.data)
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
