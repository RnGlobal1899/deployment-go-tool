package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"grc-deploy/core/logger"
	"grc-deploy/core/modules/gemcofinanceiro"
	"grc-deploy/core/modules/kaspersky"
	"grc-deploy/core/modules/kaspersky/endpointsecurity"
	"grc-deploy/core/modules/kaspersky/networkagent"
	"grc-deploy/core/modules/ocs"
	"grc-deploy/core/modules/wps"
	"grc-deploy/core/report"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	logger.InitLogger("C:\\TI_Setup_Temp")
	scannerMenu := bufio.NewScanner(os.Stdin)

	// Loop infinito iterativo para múltiplos testes sem derrubar o processo
	for {
		fmt.Println("\n============================================================")
		fmt.Println("      GRC DEPLOY - TESTES DE MÓDULOS CLI INTERATIVO")
		fmt.Println("============================================================")
		fmt.Println("  1) Kaspersky Manager (Endpoint & Network Agent)")
		fmt.Println("  2) WPS Office")
		fmt.Println("  3) OCS Inventory Agent")
		fmt.Println("  4) Gemco Financeiro")
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
			// --- MÓDULO OCS INVENTORY AGENT ---
			fmt.Println("\n--- MENU DE SELEÇÃO OCS INVENTORY ---")
			ocsManager := ocs.NewOcsManager()

			if ocsManager.CheckInstalled() {
				logger.LogWarning("O OCS Agent já está instalado nesta máquina.")
				fmt.Println("  1) Forçar Reinstalação (Modo Interativo)")
				fmt.Println("  0) Voltar ao menu principal")
				fmt.Print("Escolha uma opção: ")

				scannerMenu.Scan()
				optOcs := strings.TrimSpace(scannerMenu.Text())

				if optOcs == "1" {
					ocsManager.Deploy(true)
				} else {
					logger.WriteLog("Operação cancelada pelo usuário.", "INFO", logger.Cyan)
				}
			} else {
				logger.LogStep("OCS Agent NÃO detectado.")
				fmt.Println("  1) Iniciar Deploy (Download -> Execução Manual com GUI)")
				fmt.Println("  0) Voltar ao menu principal")
				fmt.Print("Escolha uma opção: ")

				scannerMenu.Scan()
				optOcs := strings.TrimSpace(scannerMenu.Text())

				if optOcs == "1" {
					ocsManager.Deploy(false)
				}
			}

		case "4":
			// --- MÓDULO GEMCO FINANCEIRO ---
			fmt.Println("\n--- MENU DE SELEÇÃO GEMCO FINANCEIRO ---")
			gfBaseMock := gemcofinanceiro.New(nil)
			isGfInstalled := gfBaseMock.IsInstalled()

			if isGfInstalled {
				logger.LogWarning("A base do Gemco Financeiro já está instalada nesta máquina.")
				fmt.Println("  1) Desinstalar (Clean Nuke Extremo Direcionado)")
				fmt.Println("  2) Instalar apenas SP/Custom (Atualização Isolada)")
				fmt.Println("  3) Reinstalar Completo (Nuke -> Base -> SP/Custom)")
				fmt.Println("  0) Voltar ao menu principal")
			} else {
				logger.LogStep("Base do Gemco Financeiro NÃO detectada.")
				fmt.Println("  1) Instalar apenas a BASE (Pacote_v11)")
				fmt.Println("  2) Instalar BASE + SP/Custom (Sequência Completa)")
				fmt.Println("  0) Voltar ao menu principal")
			}
			fmt.Print("Escolha uma opção: ")

			scannerMenu.Scan()
			optGf := strings.TrimSpace(scannerMenu.Text())

			if optGf == "0" || optGf == "" {
				continue
			}

			// Função auxiliar isolada para renderizar catálogo na CLI e coletar a fila via índices
			buildGfQueue := func() []string {
				var keys []string
				for k := range gemcofinanceiro.Catalog {
					keys = append(keys, k)
				}
				fmt.Println("\n  [CATÁLOGO DISPONÍVEL - GEMCO FINANCEIRO]")
				for i, k := range keys {
					fmt.Printf("   [%d] %s\n", i, k)
				}
				fmt.Print("  Digite os números desejados separados por vírgula (ex: 0,2) ou Enter para pular: ")
				scannerMenu.Scan()
				input := strings.TrimSpace(scannerMenu.Text())
				var queue []string
				if input != "" {
					parts := strings.Split(input, ",")
					for _, p := range parts {
						idx, err := strconv.Atoi(strings.TrimSpace(p))
						if err == nil && idx >= 0 && idx < len(keys) {
							queue = append(queue, keys[idx])
						}
					}
				}
				return queue
			}

			if isGfInstalled {
				switch optGf {
				case "1":
					gfBaseMock.CleanNuke()
					logger.WriteLog("Gemco Financeiro desinstalado com sucesso.", "SUCCESS", logger.Green)
				case "2":
					q := buildGfQueue()
					mod := gemcofinanceiro.New(q)
					var wg sync.WaitGroup
					wg.Add(1)
					go mod.Download(false, &wg)
					wg.Wait()
					mod.Install(false)
				case "3":
					q := buildGfQueue()
					mod := gemcofinanceiro.New(q)
					var wg sync.WaitGroup
					wg.Add(1)
					go mod.Download(true, &wg)
					wg.Wait()
					mod.Install(true)
				default:
					logger.WriteLog("Opção inválida.", "ERROR", logger.Red)
				}
			} else {
				switch optGf {
				case "1":
					mod := gemcofinanceiro.New(nil)
					var wg sync.WaitGroup
					wg.Add(1)
					go mod.Download(true, &wg)
					wg.Wait()
					mod.Install(true)
				case "2":
					q := buildGfQueue()
					mod := gemcofinanceiro.New(q)
					var wg sync.WaitGroup
					wg.Add(1)
					go mod.Download(true, &wg)
					wg.Wait()
					mod.Install(true)
				default:
					logger.WriteLog("Opção inválida.", "ERROR", logger.Red)
				}
			}
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
