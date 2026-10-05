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
)

//go:embed all:frontend/dist
var assets embed.FS

// Names of the two events that let the screen save its last edits before the window closes.
const (
	eventFlush   = "typewriter:flush"
	eventFlushed = "typewriter:flushed"
)

func main() {
	store, err := OpenStore()
	if err != nil {
		showStartupError(err)
		return
	}

	store.StartAutoBackup() // one copy of your data per day

	var win *application.WebviewWindow
	app := application.New(application.Options{
		Name:        "Typewriter",
		Description: "Write books. Track worlds, people and paths.",
		Services: []application.Service{
			application.NewService(store),
			application.NewService(&Exporter{store: store}),
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

	win = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Typewriter",
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(17, 22, 29),
		URL:              "/",
	})

	// Closing the window: first ask the screen to save what is still waiting, then close.
	// If the screen does not answer within 3 seconds, close anyway.
	var saved, asking atomic.Bool
	app.Event.On(eventFlushed, func(*application.CustomEvent) {
		saved.Store(true)
		win.Close()
	})
	win.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
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
