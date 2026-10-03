package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	store, err := OpenStore()
	if err != nil {
		log.Fatal(err)
	}

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
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Typewriter",
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(24, 22, 28),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
