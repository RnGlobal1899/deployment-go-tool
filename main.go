package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"strings"
	"sync"

	"grc-deploy/core/logger"
	"grc-deploy/core/modules/kaspersky"
	"grc-deploy/core/modules/kaspersky/endpointsecurity"
	"grc-deploy/core/modules/kaspersky/networkagent"
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

	// --- MÓDULO KASPERSKY MANAGER (INTERATIVO) ---
	fmt.Println("\n--- MENU DE SELEÇÃO KASPERSKY ---")
	fmt.Println("1) Gerenciar apenas o Network Agent")
	fmt.Println("2) Gerenciar apenas o Endpoint Security")
	fmt.Println("3) Gerenciar Ambos (Agente -> Endpoint)")
	fmt.Println("0) Pular teste do módulo Kaspersky")
	fmt.Print("Escolha uma opção: ")

	scannerKasp := bufio.NewScanner(os.Stdin)
	scannerKasp.Scan()
	opcaoKasp := strings.TrimSpace(scannerKasp.Text())

	kaspManager := kaspersky.NewKasperskyManager()

	// Processa Agente
	if opcaoKasp == "1" || opcaoKasp == "3" {
		fmt.Println("\n  [KASPERSKY] --- GERENCIANDO NETWORK AGENT ---")
		if networkagent.CheckInstalled() {
			logger.LogWarning("O Network Agent já está instalado nesta máquina.")
			fmt.Println("  1) Desinstalar")
			fmt.Println("  2) Reapontar servidor (klmover)")
			fmt.Println("  3) Seguir em frente (Pular)")
			fmt.Print("Escolha uma opção: ")
			scannerKasp.Scan()
			optAgt := strings.TrimSpace(scannerKasp.Text())

			if optAgt == "1" {
				kaspManager.UninstallNetworkAgent()
			} else if optAgt == "2" {
				fmt.Print("Informe o IP do servidor KSC: ")
				scannerKasp.Scan()
				ip := strings.TrimSpace(scannerKasp.Text())
				kaspManager.RepointNetworkAgent(ip)
			} else {
				logger.WriteLog("Etapa do Network Agent pulada pelo usuário.", "INFO", logger.Cyan)
			}
		} else {
			logger.LogStep("Network Agent NÃO detectado.")
			fmt.Print("Informe o IP do servidor KSC para a instalação: ")
			scannerKasp.Scan()
			ip := strings.TrimSpace(scannerKasp.Text())

			logger.LogStep("Iniciando fase assíncrona (Download Agent)...")
			var wg sync.WaitGroup
			wg.Add(1)
			go networkagent.DownloadAsync(&wg)
			wg.Wait()

			logger.LogStep("Iniciando instalação do Agente...")
			networkagent.InstallOrRepoint(ip)
		}
	}

	// Processa Endpoint
	if opcaoKasp == "2" || opcaoKasp == "3" {
		fmt.Println("\n  [KASPERSKY] --- GERENCIANDO ENDPOINT SECURITY ---")
		if endpointsecurity.CheckInstalled() {
			logger.LogWarning("O Endpoint Security já está instalado nesta máquina.")
			fmt.Println("  1) Desinstalar")
			fmt.Println("  2) Seguir em frente (Pular)")
			fmt.Print("Escolha uma opção: ")
			scannerKasp.Scan()
			optEnd := strings.TrimSpace(scannerKasp.Text())

			if optEnd == "1" {
				err := kaspManager.UninstallEndpoint("", "")
				if err == endpointsecurity.ErrPasswordRequired {
					logger.LogWarning("Desinstalação bloqueada. É necessário inserir as credenciais KLAdmin.")
					fmt.Print("Usuário KLAdmin: ")
					scannerKasp.Scan()
					user := strings.TrimSpace(scannerKasp.Text())
					fmt.Print("Senha KLAdmin: ")
					scannerKasp.Scan()
					pass := strings.TrimSpace(scannerKasp.Text())

					errRetry := kaspManager.UninstallEndpoint(user, pass)
					if errRetry != nil {
						logger.WriteLog(fmt.Sprintf("Falha mesmo com credenciais: %v", errRetry), "ERROR", logger.Red)
					}
				} else if err != nil {
					logger.WriteLog(fmt.Sprintf("Erro ao desinstalar Endpoint: %v", err), "ERROR", logger.Red)
				}
			} else {
				logger.WriteLog("Etapa do Endpoint Security pulada pelo usuário.", "INFO", logger.Cyan)
			}
		} else {
			logger.LogStep("Endpoint Security NÃO detectado.")
			fmt.Print("Informe a licença de ativação (ou deixe em branco para instalar sem ativar): ")
			scannerKasp.Scan()
			lic := strings.TrimSpace(scannerKasp.Text())

			logger.LogStep("Iniciando fase assíncrona (Download Endpoint)...")
			var wg sync.WaitGroup
			wg.Add(1)
			go endpointsecurity.DownloadAsync(&wg)
			wg.Wait()

			logger.LogStep("Iniciando instalação do Endpoint...")
			endpointsecurity.InstallOrActivate(lic)
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
