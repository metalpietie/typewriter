package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

//go:embed all:frontend/dist
var assets embed.FS

// Names of the two events that let the screen save its last edits before the window closes.
const (
	eventFlush   = "typewriter:flush"
	eventFlushed = "typewriter:flushed"
)

func main() {
	// During an update the app starts a copy of itself that only swaps the program file. That copy must not open
	// the data or start a backup, so it is sent off here, before anything else (it never returns in that case).
	updater.HandleHelperMode()

	store, err := OpenStore()
	if err != nil {
		showStartupError(err)
		return
	}

	store.StartAutoBackup() // one copy of your data per day
	noteInstalledVersion()  // Settings > Apps shows the right version after the app updated itself
	removeOldCopy()         // the old program file that an update left behind

	phone := NewPhoneSync(store, nil) // notes from phones, over the Wi-Fi. It listens only when it is switched on.
	defer phone.shutdown()

	updates := &Updates{store: store} // new versions from GitHub (updates.go)

	var win *application.WebviewWindow
	app := application.New(application.Options{
		Name:        "Typewriter",
		Description: "Write books. Track worlds, people and paths.",
		Services: []application.Service{
			application.NewService(store),
			application.NewService(&Exporter{store: store}),
			application.NewService(phone),
			application.NewService(updates),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		// Two copies would share one data file and overwrite each other. A second start only shows the first window.
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.typewriter.app",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if win != nil {
					win.Restore()
					win.Focus()
				}
			},
		},
	})

	phone.emit = func(name string) { app.Event.Emit(name) } // a phone changed notes: the screen reads them again
	phone.resume()
	updates.init(app)

	// the window opens maximised when it was maximised at the last close (windowstate.go)
	stateFile := windowStateFile()
	start := application.WindowStateNormal
	if loadWindowState(stateFile).Maximised {
		start = application.WindowStateMaximised
	}
	win = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Typewriter",
		StartState:       start,
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(17, 22, 29),
		URL:              "/",
	})

	// Closing the window: first ask the screen to save what is still waiting, then close.
	// If the screen does not answer within 3 seconds, close anyway.
	var saved, asking atomic.Bool
	updates.allowClose = func() { saved.Store(true) }
	app.Event.On(eventFlushed, func(*application.CustomEvent) {
		saved.Store(true)
		// a new version is ready and should start when the window closes: it closes the app itself
		if updates.wantsRestartOnClose() && updates.RestartToUpdate() == nil {
			return
		}
		win.Close()
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		if !win.IsMinimised() { // a minimised window does not tell if it was maximised: keep what was saved
			saveWindowState(stateFile, windowState{Maximised: win.IsMaximised()})
		}
		if saved.Load() {
			return
		}
		e.Cancel()
		if !asking.CompareAndSwap(false, true) {
			return
		}
		win.EmitEvent(eventFlush)
		go func() {
			time.Sleep(3 * time.Second)
			saved.Store(true)
			win.Close()
		}()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// showStartupError tells the user why the app cannot open. The app has no console window,
// so without this it would simply disappear.
func showStartupError(cause error) {
	log.Println("could not open the data:", cause)
	where := "the Typewriter folder"
	if dir, err := os.UserConfigDir(); err == nil {
		where = filepath.Join(dir, "typewriter")
	}
	msg := fmt.Sprintf("Typewriter could not open your data.\n\n%v\n\nNothing was changed or deleted.\nYour data is in: %s\nDaily copies are in the folder \"backups\" inside it. "+
		"To go back to a copy, close the app, then copy the newest auto-… file over typewriter.db.", cause, where)
	app := application.New(application.Options{Name: "Typewriter"})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		d := app.Dialog.Error().SetTitle("Typewriter cannot start").SetMessage(msg)
		d.AddButton("Close").OnClick(func() { app.Quit() })
		d.Show()
	})
	if err := app.Run(); err != nil {
		log.Println(err)
	}
}
