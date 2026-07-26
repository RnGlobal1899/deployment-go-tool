package main

import (
	"embed"
	"fmt"
	"os"
	"sync"

	"grc-deploy/core/logger"
	"grc-deploy/core/modules/anydesk"
	"grc-deploy/core/modules/chrome"
	"grc-deploy/core/modules/firefox"
	"grc-deploy/core/modules/vnc"
	"grc-deploy/core/modules/winrar"
	"grc-deploy/core/report"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	fmt.Println(">> Iniciando teste de software via terminal")

	// Inicializa o sistema de logs e ponteiros
	logger.InitLogger("C:\\TI_Setup_Temp")
	logger.WriteLog("Sessão de teste VNC iniciada...", "INFO", logger.Cyan)

	tempDir := "C:\\TI_Setup_Temp"
	vnc.Deploy(tempDir)

	// --- MÓDULO UTILITÁRIOS ESSENCIAIS ---
	logger.LogStep("Iniciando Módulo de Utilitários Essenciais...")

	chromeInst := chrome.New(tempDir)
	firefoxInst := firefox.New(tempDir)
	anydeskInst := anydesk.New(tempDir)
	winrarInst := winrar.New(tempDir)

	// FASE DE REDE: Concorrência massiva via WaitGroup
	logger.LogStep("Iniciando fase assíncrona (Downloads Paralelos)...")
	var wg sync.WaitGroup
	wg.Add(4)

	go chromeInst.Download(&wg)
	go firefoxInst.Download(&wg)
	go anydeskInst.Download(&wg)
	go winrarInst.Download(&wg) // O fallback já é baixado por prevenção

	wg.Wait()
	logger.LogSuccess("Fase de Rede concluída.")

	// FASE DE INSTALAÇÃO: Estritamente Sequencial (Respeitando o bloqueio do Windows Installer/MSI)
	logger.LogStep("Iniciando fase de instalação sequencial (Zero Touch)...")
	chromeInst.Install()
	firefoxInst.Install()
	anydeskInst.Install()
	winrarInst.Install()

	logger.LogSuccess("Módulo Utilitários Essenciais finalizado com sucesso.")

	os.Exit(0)

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
