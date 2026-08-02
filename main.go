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
	"grc-deploy/core/modules/wps"
	"grc-deploy/core/report"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	logger.InitLogger("C:\\TI_Setup_Temp\\deployment_tool.txt")
	scannerMenu := bufio.NewScanner(os.Stdin)

	// Loop infinito iterativo para múltiplos testes sem derrubar o processo
	for {
		fmt.Println("\n============================================================")
		fmt.Println("      GRC DEPLOY - TESTES DE MÓDULOS CLI INTERATIVO")
		fmt.Println("============================================================")
		fmt.Println("  1) Kaspersky Manager (Endpoint & Network Agent)")
		fmt.Println("  2) WPS Office")
		fmt.Println("  3) OCS Inventory Agent (Futuro)")
		fmt.Println("  0) Sair do Loop CLI (Prosseguir para UI ou Encerrar)")
		fmt.Print("  Escolha uma opção: ")

		scannerMenu.Scan()
		mainOpt := strings.TrimSpace(scannerMenu.Text())

		if mainOpt == "0" {
			logger.LogStep("Saindo do Menu Iterativo CLI...")
			break
		}

		switch mainOpt {
		case "1":
			// --- MÓDULO KASPERSKY MANAGER ---
			fmt.Println("\n--- MENU DE SELEÇÃO KASPERSKY ---")
			fmt.Println("1) Gerenciar apenas o Network Agent")
			fmt.Println("2) Gerenciar apenas o Endpoint Security")
			fmt.Println("3) Gerenciar Ambos (Agente -> Endpoint)")
			fmt.Println("0) Voltar ao menu principal")
			fmt.Print("Escolha uma opção: ")

			scannerMenu.Scan()
			opcaoKasp := strings.TrimSpace(scannerMenu.Text())
			kaspManager := kaspersky.NewKasperskyManager()

			if opcaoKasp == "1" || opcaoKasp == "3" {
				fmt.Println("\n  [KASPERSKY] --- GERENCIANDO NETWORK AGENT ---")
				if networkagent.CheckInstalled() {
					logger.LogWarning("O Network Agent já está instalado nesta máquina.")
					fmt.Println("  1) Desinstalar")
					fmt.Println("  2) Reapontar servidor (klmover)")
					fmt.Println("  3) Seguir em frente (Pular)")
					fmt.Print("Escolha uma opção: ")
					scannerMenu.Scan()
					optAgt := strings.TrimSpace(scannerMenu.Text())

					if optAgt == "1" {
						kaspManager.UninstallNetworkAgent()
					} else if optAgt == "2" {
						fmt.Print("Informe o IP do servidor KSC: ")
						scannerMenu.Scan()
						ip := strings.TrimSpace(scannerMenu.Text())
						kaspManager.RepointNetworkAgent(ip)
					} else {
						logger.WriteLog("Etapa do Network Agent pulada pelo usuário.", "INFO", logger.Cyan)
					}
				} else {
					logger.LogStep("Network Agent NÃO detectado.")
					fmt.Print("Informe o IP do servidor KSC para a instalação: ")
					scannerMenu.Scan()
					ip := strings.TrimSpace(scannerMenu.Text())

					logger.LogStep("Iniciando fase assíncrona (Download Agent)...")
					var wg sync.WaitGroup
					wg.Add(1)
					go networkagent.DownloadAsync(&wg)
					wg.Wait()

					logger.LogStep("Iniciando instalação do Agente...")
					networkagent.InstallOrRepoint(ip)
				}
			}

			if opcaoKasp == "2" || opcaoKasp == "3" {
				fmt.Println("\n  [KASPERSKY] --- GERENCIANDO ENDPOINT SECURITY ---")
				if endpointsecurity.CheckInstalled() {
					logger.LogWarning("O Endpoint Security já está instalado nesta máquina.")
					fmt.Println("  1) Desinstalar")
					fmt.Println("  2) Seguir em frente (Pular)")
					fmt.Print("Escolha uma opção: ")
					scannerMenu.Scan()
					optEnd := strings.TrimSpace(scannerMenu.Text())

					if optEnd == "1" {
						err := kaspManager.UninstallEndpoint("", "")
						if err == endpointsecurity.ErrPasswordRequired {
							logger.LogWarning("Desinstalação bloqueada. É necessário inserir as credenciais KLAdmin.")
							fmt.Print("Usuário KLAdmin: ")
							scannerMenu.Scan()
							user := strings.TrimSpace(scannerMenu.Text())
							fmt.Print("Senha KLAdmin: ")
							scannerMenu.Scan()
							pass := strings.TrimSpace(scannerMenu.Text())

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
					scannerMenu.Scan()
					lic := strings.TrimSpace(scannerMenu.Text())

					logger.LogStep("Iniciando fase assíncrona (Download Endpoint)...")
					var wg sync.WaitGroup
					wg.Add(1)
					go endpointsecurity.DownloadAsync(&wg)
					wg.Wait()

					logger.LogStep("Iniciando instalação do Endpoint...")
					endpointsecurity.InstallOrActivate(lic)
				}
			}

		case "2":
			// --- MÓDULO WPS OFFICE ---
			fmt.Println("\n--- MENU DE SELEÇÃO WPS OFFICE ---")
			wpsManager := wps.NewWpsManager()

			isInstalled, uninstPath := wpsManager.CheckInstalled()
			if isInstalled {
				logger.LogWarning("O WPS Office já está instalado nesta máquina.")
				fmt.Println("  1) Desinstalar silenciosamente")
				fmt.Println("  2) Forçar Reinstalação Completa")
				fmt.Println("  0) Voltar ao menu principal")
				fmt.Print("Escolha uma opção: ")

				scannerMenu.Scan()
				optWps := strings.TrimSpace(scannerMenu.Text())

				if optWps == "1" {
					wpsManager.Uninstall(uninstPath)
				} else if optWps == "2" {
					wpsManager.Deploy(true)
				} else {
					logger.WriteLog("Operação cancelada pelo usuário.", "INFO", logger.Cyan)
				}
			} else {
				logger.LogStep("WPS Office NÃO detectado.")
				fmt.Println("  1) Iniciar Deploy Autônomo (Download -> Instalação Silenciosa)")
				fmt.Println("  0) Voltar ao menu principal")
				fmt.Print("Escolha uma opção: ")

				scannerMenu.Scan()
				optWps := strings.TrimSpace(scannerMenu.Text())

				if optWps == "1" {
					wpsManager.Deploy(false)
				}
			}

		case "3":
			logger.LogWarning("Módulo OCS Inventory ainda não implementado. (Próxima etapa!)")

		default:
			logger.WriteLog("Opção inválida.", "ERROR", logger.Red)
		}
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
