// Typewriter: write books, and track worlds, people and paths.
// The app is in internal/app. This file only puts the built screen (frontend/dist) into the program.
package main

import (
	"embed"

	"typewriter/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app.Run(assets)
}
