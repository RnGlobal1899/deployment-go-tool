package main

import (
	"embed"
	"log"

	"grc-deploy/core/logger"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Inicializa o motor de logs
	logger.InitLogger(`C:\TI_Setup_Temp`) // Define o diretório temporário para armazenar os logs

	// Cria uma instância da estrutura App (que reside no app.go)
	app := NewApp()

	// Inicializa a aplicação Wails / WebView
	err := wails.Run(&options.App{
		Title:  "GRC Deploy",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 1}, // Tonalidade Dark (bg-slate-900)
		OnStartup:        app.startup,
		Bind: []interface{}{
			app, // Expõe os métodos do backend para o frontend (API-First)
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
