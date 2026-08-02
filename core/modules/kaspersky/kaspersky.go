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
