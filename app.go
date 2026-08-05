package main

import (
	"context"

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
