package kaspersky

import (
	"fmt"
	"grc-deploy/core/modules/kaspersky/endpointsecurity"
	"grc-deploy/core/modules/kaspersky/networkagent"
	"os"
	"os/exec"
	"strings"
	"sync"

	"grc-deploy/core/logger"
)

// Exporta os métodos para o Data Binding com o frontend
type KasperskyManager struct{}

// Inicializa o KasperskyManager
func NewKasperskyManager() *KasperskyManager {
	return &KasperskyManager{}
}

// Implementa a interface GRCModule para o KasperskyManager
func (k *KasperskyManager) GetID() string          { return "kaspersky" }
func (k *KasperskyManager) GetName() string        { return "Kaspersky Endpoint" }
func (k *KasperskyManager) GetDescription() string { return "Endpoint Security e Network Agent." }
func (k *KasperskyManager) GetIconSVG() string {
	return `<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" /></svg>`
}
func (k *KasperskyManager) RunSilent() error {
	return k.DeployKasperskyModules("", "")
}

// Organiza a esteira de execução dos módulos do Kaspersky
func (k *KasperskyManager) DeployKasperskyModules(serverIP, activationcode string) error {
	logger.LogStep("Iniciando Kaspersky Manager...")

	var wg sync.WaitGroup

	// Fase 1: Downloads (Assincrona)
	logger.LogStep("Fase 1: Iniciando downloads dos módulos do Kaspersky...")
	wg.Add(2)
	go networkagent.DownloadAsync(&wg)
	go endpointsecurity.DownloadAsync(&wg)

	wg.Wait()
	logger.LogStep("Fase 1: Downloads concluídos.")

	// Fase 2: Instalação (Sequencial)
	logger.LogStep("Fase de instalação do Network Agent ...")
	errAgent := networkagent.InstallOrRepoint(serverIP)
	if errAgent != nil {
		logger.WriteLog(fmt.Sprintf("Erro na instalação do Network Agent: %v", errAgent), "ERROR", logger.Red)
		return errAgent
	}

	logger.LogStep("Fase de instalação do Endpoint Security ...")
	errEndpoint := endpointsecurity.InstallOrActivate(activationcode)
	if errEndpoint != nil {
		logger.WriteLog(fmt.Sprintf("Erro na instalação do Endpoint Security: %v", errEndpoint), "ERROR", logger.Red)
		return errEndpoint
	}

	logger.LogStep("Kaspersky Manager finalizado com sucesso.")
	return nil
}

// Executa validações avançadas (Serviço do Windows + Múltiplos Paths)
func (k *KasperskyManager) CheckStatus(component string) bool {
	if component == "agent" {
		// 1. Verificação de Serviço Ativo
		cmd := exec.Command("cmd", "/C", "sc query klnagent")
		out, err := cmd.CombinedOutput()
		if err == nil && (strings.Contains(string(out), "RUNNING") || strings.Contains(string(out), "STOPPED")) {
			return true
		}

		// 2. Verificação de Paths (Fallbacks)
		paths := []string{
			`C:\Program Files (x86)\Kaspersky Lab\NetworkAgent\klnagent.exe`,
			`C:\Program Files\Kaspersky Lab\NetworkAgent\klnagent.exe`,
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	} else if component == "endpoint" {
		// 1. Verificação de Serviço Ativo
		cmd := exec.Command("cmd", "/C", "sc query avp")
		out, err := cmd.CombinedOutput()
		if err == nil && (strings.Contains(string(out), "RUNNING") || strings.Contains(string(out), "STOPPED")) {
			return true
		}

		// 2. Verificação de Paths (Adicionado variação da imagem)
		paths := []string{
			`C:\Program Files (x86)\Kaspersky Lab\Kaspersky Endpoint Security for Windows\avp.exe`,
			`C:\Program Files (x86)\Kaspersky Lab\KES\avp.exe`,
			`C:\Program Files\Kaspersky Lab\KES\avp.exe`,
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	}
	return false
}

// ExecuteWizard recebe a ordem do frontend e roteia para as funções corretas
func (k *KasperskyManager) ExecuteWizard(agentAction, agentPayload, kesAction, kesPayload string) {
	logger.LogStep("Iniciando Esteira Kaspersky (Modo Avançado)...")

	if agentAction != "" && agentAction != "skip" {
		logger.WriteLog(fmt.Sprintf("Executando no Agente: %s", agentAction), "INFO", logger.Cyan)
		if agentAction == "repoint" {
			k.RepointNetworkAgent(agentPayload)
		} else if agentAction == "uninstall" {
			k.UninstallNetworkAgent()
		} else if agentAction == "install" || agentAction == "reinstall" {
			if agentAction == "reinstall" {
				k.UninstallNetworkAgent()
			}
			networkagent.InstallOrRepoint(agentPayload)
		}
	}

	if kesAction != "" && kesAction != "skip" {
		logger.WriteLog(fmt.Sprintf("Executando no Endpoint: %s", kesAction), "INFO", logger.Cyan)
		if kesAction == "activate" {
			endpointsecurity.InstallOrActivate(kesPayload)
		} else if kesAction == "uninstall" {
			// Credenciais default de KLAdmin - devem ser tratadas futuramente
			k.UninstallEndpoint("", "")
		} else if kesAction == "install" || kesAction == "reinstall" {
			endpointsecurity.InstallOrActivate(kesPayload)
		}
	}
	logger.LogSuccess("Processamento Kaspersky concluído com sucesso.")
}

// Funções exportadas para UI

// RepointNetworkAgent força o redirecionamento imediato de instâncias órfãs.
func (k *KasperskyManager) RepointNetworkAgent(serverIP string) error {
	return networkagent.Repoint(serverIP)
}

// UninstallEndpoint força a limpeza do Endpoint, injetando credenciais KLAdmin se existirem.
func (k *KasperskyManager) UninstallEndpoint(adminLogin, adminPass string) error {
	return endpointsecurity.Uninstall(adminLogin, adminPass)
}

// UninstallNetworkAgent erradica o agente local chamando msiexec de desinstalação.
func (k *KasperskyManager) UninstallNetworkAgent() error {
	return networkagent.CleanupResidue()
}
