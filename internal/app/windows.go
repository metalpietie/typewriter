package app

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Windows opens more windows of the app, for a second screen: a map, a character or the reference next to the text.
// Each window is the whole app on one screen. The windows tell each other what they saved (db.svelte.ts), and
// when the main window closes, the others save and close with it (run.go).
type Windows struct {
	app  *application.App
	mu   sync.Mutex
	open map[string]*application.WebviewWindow
	n    int
}

// tabs that a new window may open on (the screens of the menu)
var windowTabs = map[string]bool{
	"home": true, "draft": true, "write": true, "board": true, "arcs": true, "characters": true, "relationships": true,
	"world": true, "glossary": true, "factions": true, "items": true, "family": true, "checks": true, "stats": true,
	"maps": true, "timeline": true, "export": true, "sources": true, "ideas": true, "reference": true,
}

// OpenWindow opens a new window on a screen (tab), with a title such as "Maps & Paths".
func (w *Windows) OpenWindow(tab, title string) error {
	if !windowTabs[tab] {
		return fmt.Errorf("there is no screen %q", tab)
	}
	if w.app == nil {
		return fmt.Errorf("the app has not started")
	}
	w.mu.Lock()
	w.n++
	name := fmt.Sprintf("window-%d", w.n)
	if w.open == nil {
		w.open = map[string]*application.WebviewWindow{}
	}
	w.mu.Unlock()
	title = strings.TrimSpace(title)
	if title == "" {
		title = tab
	}
	win := w.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             name,
		Title:            "Typewriter · " + title,
		Width:            1100,
		Height:           780,
		BackgroundColour: application.NewRGB(12, 17, 24),
		URL:              "/?window=" + url.QueryEscape(name) + "&tab=" + url.QueryEscape(tab),
	})
	w.mu.Lock()
	w.open[name] = win
	w.mu.Unlock()
	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		w.mu.Lock()
		delete(w.open, name)
		w.mu.Unlock()
	})
	win.Show()
	return nil
}

// count is the number of extra windows that are open.
func (w *Windows) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.open)
}

// closeAll closes the extra windows (the main window is closing).
func (w *Windows) closeAll() {
	w.mu.Lock()
	list := make([]*application.WebviewWindow, 0, len(w.open))
	for _, win := range w.open {
		list = append(list, win)
	}
	w.open = map[string]*application.WebviewWindow{}
	w.mu.Unlock()
	for _, win := range list {
		win.Close()
	}
}
