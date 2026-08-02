package networkagent

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"grc-deploy/core/downloader"
	"grc-deploy/core/executor"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"
)

const (
	AgentURL   = "https://www.dropbox.com/scl/fi/lhw6llulzpeevsjiyjkoa/Kaspersky-Network-Agent.msi?rlkey=e1uppsb7ycxnh2j6xclbffiro&st=755d75h4&dl=1"
	InstallDir = "C:\\Program Files (x86)\\Kaspersky Lab\\NetworkAgent"
	TempDir    = "C:\\TI_Setup_Temp\\Kaspersky"
	AgentMSI   = "C:\\TI_Setup_Temp\\Kaspersky\\NetworkAgent.msi"
	LogFile    = "C:\\TI_Setup_Temp\\Kaspersky\\agent_install.log"
)

// DownloadAsync baixa o instalador do Agente isolando o bloqueio de I/O em sua própria goroutine.
func DownloadAsync(wg *sync.WaitGroup) {
	defer wg.Done()
	if CheckCached() {
		logger.LogStep("Network Agent já está em cache. Pulando download.")
		report.AddDeployReport("Network Agent", "Download", "SUCESSO", "Instalador já em cache.")
		return
	}

	var fila []downloader.DownloadItem

	os.MkdirAll(TempDir, os.ModePerm)
	fila = append(fila, downloader.DownloadItem{
		URL:         AgentURL,
		Destination: filepath.Join(TempDir, "NetworkAgent.msi"),
		Label:       "Network Agent",
		ExpectedMB:  40.0,
		MagicType:   "MSI",
	})

	err := downloader.DownloadsParalelos(fila)

	if err != nil {
		logger.WriteLog(fmt.Sprintf("Falha ao baixar o Network Agent: %v", err), "ERROR", logger.Red)
		report.AddDeployReport("Network Agent", "Download", "FALHA", fmt.Sprintf("Erro: %v", err))
	}
}

// Valida se o MSI já foi baixado com tamanho estrutural coerente.
func CheckCached() bool {
	info, err := os.Stat(AgentMSI)
	return err == nil && info.Size() > 5000000 // Aprox > 5MB
}

// Valida a presença do diretório binário físico.
func CheckInstalled() bool {
	_, err := os.Stat(InstallDir)
	return !os.IsNotExist(err)
}

// Implementa a inteligência do fluxo: Redireciona se existir, Instala se não existir.
func InstallOrRepoint(serverIP string) error {
	if serverIP == "" {
		logger.LogStep("IP do servidor Kaspersky não fornecido")
		return nil
	}

	if CheckInstalled() {
		logger.LogStep("Network Agent já instalado. Acionando redirecionamento...")
		return Repoint(serverIP)
	}

	return Install(serverIP)
}

// Executa o processo de instalação via msiexec
func Install(serverIP string) error {
	logger.LogStep(fmt.Sprintf("Iniciando instalação do Network Agent para o servidor %s...", serverIP))

	// Limpeza preventina de GUIDs fantamas no regedit
	logger.LogStep("Limpando resíduos de instalações anteriores (se existirem)...")
	CleanupResidue()

	// Execução restrita da API Win32 encapsulada, com EULA injetado e interface suprimida (/qn).
	logger.LogStep("Executando msiexec.exe de forma silenciosa...")
	exitCode, err := executor.RunSilent("msiexec.exe", "/i", AgentMSI, fmt.Sprintf("SERVERADDRESS=%s", serverIP), "EULA=1", "/qn", "/norestart", "/L*v", LogFile)
	if err != nil || (exitCode != 0 && exitCode != 3010) {
		report.AddDeployReport("Kaspersky Agent", "Instalação", "Falha", fmt.Sprintf("ExitCode: %d", exitCode))
		return fmt.Errorf("msiexec (Agente) abortou a operação: %v (ExitCode %d)", err, exitCode)
	}

	logger.LogStep("Verificando se o serviço klnagent foi registrado no Windows...")
	svcCheck := `if (Get-Service -Name "klnagent" -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }`
	svcExit, _ := executor.RunSilent("powershell.exe", "-NoProfile", "-Command", svcCheck)

	if svcExit == 0 {
		logger.WriteLog("Kaspersky Network Agent instalado e serviço klnagent confirmado.", "SUCCESS", logger.Green)
	} else {
		logger.WriteLog("Serviço klnagent não detectado. Verifique o log se necessário.", "WARNING", logger.Yellow)
	}

	report.AddDeployReport("Kaspersky Agent", "Instalação", "Sucesso", "Agente configurado e sincronizado com "+serverIP)
	return nil
}

// Chama o klmover.exe para forçar a comunicação de rede KSC.
func Repoint(serverIP string) error {
	klmoverPath := filepath.Join(InstallDir, "klmover.exe")
	if _, err := os.Stat(klmoverPath); os.IsNotExist(err) {
		return fmt.Errorf("binário klmover.exe não encontrado no diretório: %s", InstallDir)
	}

	_, err := executor.RunSilent(klmoverPath, "-address", serverIP, "-pn", "14000", "-ps", "13000")
	if err != nil {
		report.AddDeployReport("Kaspersky Agent", "Repontamento", "Falha", err.Error())
		return fmt.Errorf("falha interna no klmover.exe: %v", err)
	}

	report.AddDeployReport("Kaspersky Agent", "Repontamento", "Sucesso", "KSC Repontado para IP "+serverIP)
	return nil
}

// Consulta o registro para varrer resíduos do agente corrompido
func CleanupResidue() error {
	logger.LogStep("Varrendo registro em busca de resíduos do Network Agent...")

	psCmd := `
	$apps = Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*", "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*" -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -like "Agente de Rede*" }
	foreach ($app in $apps) {
		$guid = $app.PSChildName
		Start-Process "msiexec.exe" -ArgumentList "/x ` + "`\"$guid`\"" + ` /qn /norestart" -Wait -NoNewWindow
	}`

	_, err := executor.RunSilent("powershell.exe", "-NoProfile", "-Command", psCmd)
	if err == nil {
		logger.WriteLog("Limpeza do Network Agent finalizada.", "INFO", logger.Cyan)
		report.AddDeployReport("Kaspersky Agent", "Limpeza de Resíduos", "Sucesso", "Resíduos do Network Agent removidos.")
	} else {
		logger.WriteLog("Falha ao limpar resíduos do Network Agent.", "ERROR", logger.Red)
		report.AddDeployReport("Kaspersky Agent", "Limpeza de Resíduos", "Falha", "Falha ao limpar resíduos do Network Agent.")
	}

	return err
}
