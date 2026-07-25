package main

import (
	"embed"
	"fmt"
	"grc-deploy/core/downloader"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	fmt.Println(">> Iniciando download via CLI")

	fila := []downloader.DownloadItem{
		{
			URL:         "https://download.anydesk.com/AnyDesk.exe",
			Destination: "C:\\TI_Setup_Temp\\Teste_Go\\AnyDesk.exe",
			Label:       "AnyDesk",
			ExpectedMB:  3.0,
			MagicType:   "EXE",
		},
		{
			URL:         "https://download.mozilla.org/?product=firefox-latest-ssl&os=win64&lang=pt-BR",
			Destination: "C:\\TI_Setup_Temp\\Teste_Go\\Firefox.exe",
			Label:       "Firefox",
			ExpectedMB:  50.0,
			MagicType:   "EXE",
		},
	}

	if err := downloader.DownloadsParalelos(fila); err != nil {
		fmt.Printf("Erro ao realizar downloads: %v\n", err)
	} else {
		fmt.Println("Downloads concluídos com sucesso! Bloqueando start do Wails")
		os.Exit(1)
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "grc-deploy",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
