package main

import (
	"embed"
	"fmt"
	"os"
	"sync"

	"grc-deploy/core/logger"
	"grc-deploy/core/modules/gemco"
	"grc-deploy/core/report"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	// Inicializa o sistema de logs e ponteiros
	logger.InitLogger("C:\\TI_Setup_Temp")
	// tempDir := "C:\\TI_Setup_Temp"
	logger.WriteLog("Iniciando sessão de teste", "INFO", logger.Cyan)

	fmt.Println("\n--- MENU DE SELEÇÃO GEMCO 2002 ---")
	fmt.Println("1) Instalar apenas a Base")
	fmt.Println("2) Instalar Base + SP 121 + Custom 11-86 (Testar Downloads Paralelos)")
	fmt.Println("3) Pular teste do Gemco")
	fmt.Print("Escolha uma opção: ")

	var opcaoGemco string
	fmt.Scanln(&opcaoGemco)

	var gemcoQueue []string
	if opcaoGemco == "2" {
		gemcoQueue = []string{"Gemco2002SP44-00-121.exe", "Custom11-86.EXE"}
	}

	if opcaoGemco == "1" || opcaoGemco == "2" {
		gemcoInst := gemco.New(gemcoQueue)

		executarGemco := true
		if gemcoInst.IsInstalled() {
			logger.LogWarning("Gemco 2002 já está instalado no sistema.")
			fmt.Print("  Deseja pular a restrição, aplicar um Clean Nuke e reinstalar agora? [s/n]: ")
			var resp string
			fmt.Scanln(&resp)

			if resp == "s" || resp == "S" {
				logger.LogStep("Executando Clean Nuke isoladamente antes do teste...")
				gemcoInst.CleanNuke()
			} else {
				executarGemco = false
				logger.WriteLog("Teste do Gemco abortado pelo usuário.", "INFO", logger.Cyan)
			}
		}

		if executarGemco {
			logger.LogStep("Iniciando fase assíncrona (Download Gemco)...")
			var wgGemco sync.WaitGroup
			wgGemco.Add(1)

			go gemcoInst.Download(&wgGemco)
			wgGemco.Wait()
			logger.LogSuccess("Fase de Rede do Gemco concluída.")

			logger.LogStep("Iniciando fase de instalação sequencial (Gemco)...")
			err := gemcoInst.Install()

			if err != nil {
				logger.WriteLog(fmt.Sprintf("Falha ao instalar o Gemco: %v", err), "ERROR", logger.Red)
			} else {
				// Revalidação obrigatória no pós-instalação
				if gemcoInst.IsInstalled() {
					logger.LogSuccess("Gemco 2002 validado com sucesso no registro pós-instalação!")

					fmt.Print("\n  Deseja iniciar o Gconfig agora (irá pausar o terminal)? [s/n]: ")
					var respGconfig string
					fmt.Scanln(&respGconfig)
					if respGconfig == "s" || respGconfig == "S" {
						gemcoInst.InitGconfig()
					}
				} else {
					logger.WriteLog("O método Install finalizou sem erros, mas IsInstalled() retornou false.", "ERROR", logger.Red)
				}
			}
		}
	}

	// os.Exit(0)

	// --- MÓDULO UTILITÁRIOS ESSENCIAIS ---
	//logger.LogStep("Iniciando Módulos...")

	//vncInst := vnc.New(tempDir)

	// --- BLOCO INTERATIVO DE TESTE (HOMOLOGAÇÃO VNC) ---
	//if vncInst.IsInstalled() {
	//logger.LogWarning("UltraVNC detectado no sistema.")
	//fmt.Print("  Deseja testar o Clean Nuke (Desinstalação) agora? [s/n]: ")
	//var resp string
	//fmt.Scanln(&resp)

	//if resp == "s" || resp == "S" {
	//vncInst.Uninstall()
	//logger.LogSuccess("Teste de desinstalação concluído. Valide o diretório e o registro.")
	//os.Exit(0) // Interrompe a esteira para você fazer a validação manual
	//}
	//}
	// ---------------------------------------------------

	//chromeInst := chrome.New(tempDir)
	//firefoxInst := firefox.New(tempDir)
	//anydeskInst := anydesk.New(tempDir)
	//winrarInst := winrar.New(tempDir)

	// Downloads paralelos via WaitGroup
	//logger.LogStep("Iniciando fase assíncrona (Downloads Paralelos)...")
	//var wg sync.WaitGroup
	//wg.Add(5)

	//go vncInst.Download(&wg)
	//go chromeInst.Download(&wg)
	//go firefoxInst.Download(&wg)
	//go anydeskInst.Download(&wg)
	//go winrarInst.Download(&wg)

	//wg.Wait()
	//logger.LogSuccess("Fase de Rede concluída.")

	// Instalaçãoo sequencial
	//logger.LogStep("Iniciando fase de instalação sequencial (Zero Touch)...")
	//vncInst.Install()
	//chromeInst.Install()
	//firefoxInst.Install()
	//anydeskInst.Install()
	//winrarInst.Install()

	//logger.LogSuccess("Módulo Utilitários Essenciais finalizado com sucesso.")

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
