package kaspersky

import (
	"fmt"
	"grc-deploy/core/modules/kaspersky/endpointsecurity"
	"grc-deploy/core/modules/kaspersky/networkagent"
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
