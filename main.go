package main

import (
	"embed"
	"fmt"
	"os"

	"grc-deploy/core/downloader"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	fmt.Println(">> Iniciando download via CLI")

	// Inicializa o sistema de logs e ponteiros
	logger.InitLogger("C:\\TI_Setup_Temp")
	logger.WriteLog("Sessão de teste de integração iniciada.", "INFO", logger.Cyan)

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
		logger.WriteLog(fmt.Sprintf("Teste apresentou falhas: %v", err), "ERROR", logger.Red)
	}

	fmt.Println("\n  ============================================================")
	fmt.Println("  >> RELATÓRIO DE DEPLOY GERADO EM MEMÓRIA <<")
	for _, item := range report.GetReport() {
		fmt.Printf("  [%s] %s — %s: %s\n", item.Status, item.Componente, item.Tarefa, item.Detalhes)
	}
	fmt.Println("  ============================================================")

	fmt.Println("\n>> TESTE CONCLUÍDO. BLOQUEANDO START DO WAILS <<")
	os.Exit(0)

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
