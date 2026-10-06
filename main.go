package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// An optional path argument opens a device or image straight away:
	//   sudo bbsdcardsniffer /dev/disk4
	var autoOpen string
	if len(os.Args) > 1 {
		autoOpen = os.Args[1]
	}
	app := NewApp(autoOpen)

	err := wails.Run(&options.App{
		Title:     "SD Card Sniffer",
		Width:     1280,
		Height:    820,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 18, B: 24, A: 1},
		OnStartup:        app.startup,
		// Closing the window must release the device, otherwise the card
		// cannot be ejected until the process dies.
		OnShutdown: app.shutdown,
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "SD Card Sniffer",
				Message: "Read-only inspection of SD cards, including Linux filesystems.",
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
