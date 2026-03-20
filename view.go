package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

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
		v := tea.NewView("Press esc to return\n\n" + m.hdImageCache)
		v.AltScreen = true
		return v
	}

	if !m.ready {
		return tea.NewView("Loading...")
	}

	title := titleStyle.Render(fmt.Sprintf("Glimpse - %s", m.filename))

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

	selectorView := m.imageSelectorView()
	rightInner := selectorView + "\n" + m.imagePane.View()
	right := rightStyle.Render(rightInner)

	content := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	imgHint := ""
	if len(m.images) > 0 {
		imgHint = accentStyle.Render(" • ctrl+p: HD image")
	}
	status := statusStyle.Render(
		fmt.Sprintf("  esc: clear/quit • tab: switch pane • ↑↓: scroll%s", imgHint))

	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, title, content, status))
	v.AltScreen = true
	return v
}

// renderImageContent returns the image preview string for the currently
// selected image, sized to fit the image viewport.
func (m model) renderImageContent(cols, rows int) string {
	if len(m.images) == 0 {
		return "\n  No image data found"
	}
	return renderPreview(m.images[m.imageIdx].img, cols, rows)
}

// imageSelectorView returns the image selector header line for the right pane.
func (m model) imageSelectorView() string {
	if len(m.images) == 0 {
		return statusStyle.Render("  No image data")
	}
	e := m.images[m.imageIdx]
	return accentStyle.Render(fmt.Sprintf("  ▲ Image: %s (%d/%d) ▼", e.key, m.imageIdx+1, len(m.images)))
}
