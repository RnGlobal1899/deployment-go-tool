package main

import (
	"context"

	"grc-deploy/core/logger"
	"grc-deploy/core/modules"
	"grc-deploy/core/modules/gemcofinanceiro"
	"grc-deploy/core/modules/kaspersky"
	"grc-deploy/core/modules/wps"
)

// App struct mantém o contexto da aplicação Wails
type App struct {
	ctx context.Context
}

// NewApp cria uma nova instância de App
func NewApp() *App {
	return &App{}
}

// startup é chamado pelo Wails logo na inicialização
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logger.SetContext(ctx) // Acopla o motor de logs ao channel de eventos do Wails para comunicação com o frontend
}

// Estrutura de dados para transitar do Go para o Wails/TS
type UIModule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IconSvg     string `json:"iconSvg"`
}

// GetSoftwareModules é o motor dinâmico que lê as instâncias reais do Strategy
func (a *App) GetSoftwareModules() []UIModule {
	activeModules := []modules.GRCModule{
		kaspersky.NewKasperskyManager(),
		wps.NewWpsManager(),
		gemcofinanceiro.New([]string{}),
	}

	var uiList []UIModule

	// Transcreve os dados reais do backend para a ponte IPC
	for _, mod := range activeModules {
		uiList = append(uiList, UIModule{
			ID:          mod.GetID(),
			Name:        mod.GetName(),
			Description: mod.GetDescription(),
			IconSvg:     mod.GetIconSVG(),
		})
	}

	return uiList
}

// Recebe o ID do card
func (a *App) InstallSoftware(id string) {
	activeModules := []modules.GRCModule{
		kaspersky.NewKasperskyManager(),
		wps.NewWpsManager(),
		gemcofinanceiro.New([]string{}),
	}

	for _, mod := range activeModules {
		if mod.GetID() == id {
			go func(m modules.GRCModule) {
				logger.LogStep("Solicitação de instalação para: " + m.GetName())
				err := m.RunSilent()
				if err != nil {
					logger.WriteLog("Erro ao instalar: "+m.GetName()+": "+err.Error(), "ERROR", logger.Red)
				} else {
					logger.LogSuccess(m.GetName() + " instalado com sucesso.")
				}
			}(mod)
			break
		}
	}
}

// Orquestra a validação do Kaspersky através do módulo isolado
func (a *App) CheckKasperskyStatus(component string) bool {
	km := kaspersky.NewKasperskyManager()
	return km.CheckStatus(component)
}

// Orquestra o roteamento dos comandos via módulo isolado
func (a *App) ExecuteKasperskyWizard(agentAction, agentPayload, kesAction, kesPayload string) {
	// A Goroutine é obrigatória aqui para que a UI do Wails não congele aguardando o fim da instalação
	go func() {
		km := kaspersky.NewKasperskyManager()
		km.ExecuteWizard(agentAction, agentPayload, kesAction, kesPayload)
	}()
}

// Orquestra a validação de presença do WPS Office no sistema
func (a *App) CheckWpsStatus() bool {
	wm := wps.NewWpsManager()
	isInstalled, _ := wm.CheckInstalled()
	return isInstalled
}

// Orquestra o roteamento das ações do WPS via módulo isolado
func (a *App) ExecuteWpsWizard(action string) {
	go func() {
		wm := wps.NewWpsManager()
		if action == "install" {
			wm.Deploy(false) // Fase de rede + instalação
		} else if action == "reinstall" {
			wm.Deploy(true) // Força remoção + fase de rede + instalação
		} else if action == "uninstall" {
			// Busca o path do uninstaller dinamicamente
			_, uninstPath := wm.CheckInstalled()
			if uninstPath != "" {
				wm.Uninstall(uninstPath)
			}
		}
	}()
}
