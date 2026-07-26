package vnc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"grc-deploy/core/downloader"
	"grc-deploy/core/executor"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"
)

const (
	vncUrl = "https://www.dropbox.com/scl/fi/n0sbgcwcntuoyf7xmt8ye/UltraVNC_1_2_17_X64_Setup-2018.exe?rlkey=slwp4mmg0s244x76ab9d3c07y&st=cmbtbbjq&dl=1"
	iniUrl = "https://www.dropbox.com/scl/fi/a6kx1cd3ou4pz8iy9nnyo/ultravnc.ini?rlkey=ghp1u133xro3xsvurauaxb5bk&st=2brz4ruh&dl=1"
	infUrl = "https://www.dropbox.com/scl/fi/bmdeurmb3o4rjf8dnhg0b/ultravnc.inf?rlkey=cu4tfnycbzkq1zkyi89431a2v&st=vyb4pwb0&dl=1"
)

// Deploy executa a esteira de provisionamento específica do UltraVNC
func Deploy(tempFolder string) {
	logger.LogStep("Iniciando Módulo: UltraVNC Installer (Ponto Zero)")
	logger.WriteLog("Módulo UltraVNC iniciado pelo Master Deploy (Go).", "INFO", logger.Cyan)

	installPath := filepath.Join(os.Getenv("ProgramFiles"), "uvnc bvba", "UltraVNC")
	winvncExe := filepath.Join(installPath, "winvnc.exe")

	// Verificação de Instalação Existente
	if _, err := os.Stat(winvncExe); err == nil {
		logger.LogWarning("O UltraVNC já está instalado nesta máquina.")
		logger.WriteLog("UltraVNC detectado. Instalação ignorada na automação Go.", "INFO", logger.Cyan)
		report.AddDeployReport("UltraVNC", "Instalação", report.StatusAviso, "Ignorado (Já estava instalado)")
		return
	}

	// Preparação do Ambiente
	vncTempDir := filepath.Join(tempFolder, "UltraVNC")
	vncExePath := filepath.Join(vncTempDir, "UltraVNC_Setup.exe")
	vncIniPath := filepath.Join(vncTempDir, "ultravnc.ini")
	vncInfPath := filepath.Join(vncTempDir, "ultravnc.inf")

	fila := []downloader.DownloadItem{
		{URL: vncUrl, Destination: vncExePath, Label: "Instalador UltraVNC", ExpectedMB: 1.0, MagicType: "EXE"},
		{URL: iniUrl, Destination: vncIniPath, Label: "Arquivo de Conf (ultravnc.ini)", ExpectedMB: 0, MagicType: "Nenhum"},
		{URL: infUrl, Destination: vncInfPath, Label: "Arquivo de Resposta (ultravnc.inf)", ExpectedMB: 0, MagicType: "Nenhum"},
	}

	// Execução Paralela da Fase de Rede
	if err := downloader.DownloadsParalelos(fila); err != nil {
		logger.WriteLog("Falha no download dos recursos do VNC. Módulo abortado.", "ERROR", logger.Red)
		report.AddDeployReport("UltraVNC", "Download", report.StatusFalha, "Erro ao baixar arquivos dependentes")
		return
	}

	// Injeção Prévia do .ini no diretório de instalação antes do setup
	logger.LogStep("Preparando ambiente e executando instalação silenciosa...")
	os.MkdirAll(installPath, os.ModePerm)
	copyFile(vncIniPath, filepath.Join(installPath, "ultravnc.ini"))
	logger.LogStep("Arquivo ultravnc.ini pré-injetado com sucesso.")

	// Instalação Sequencial Estrita
	installArgs := []string{
		"/VERYSILENT",
		fmt.Sprintf(`/LOADINF=%s`, vncInfPath),
		`/TASKS=installservice`,
		"/NORESTART",
	}

	exitCode, err := executor.RunSilent(vncExePath, installArgs...)
	if err != nil && exitCode != 0 && exitCode != 3010 {
		logger.LogWarning(fmt.Sprintf("Instalador retornou erro: %d", exitCode))
		logger.WriteLog("Falha na instalacao do UltraVNC.", "ERROR", logger.Red)
		report.AddDeployReport("UltraVNC", "Instalação", report.StatusFalha, fmt.Sprintf("ExitCode %d", exitCode))
		return
	}

	logger.LogSuccess("Instalação base concluída e serviço registrado pelo desinstalador.")

	// Ativação do Serviço e Persistência
	logger.LogStep("Ativando serviço e persistência do ícone...")
	if _, err := os.Stat(winvncExe); err == nil {
		// Inicia serviço
		executor.RunSilent("sc", "start", "uvnc_service")

		// Registro via comando nativo para persistência do Tray Icon
		regArgs := []string{"add", `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "/v", "UltraVNC", "/t", "REG_SZ", "/d", fmt.Sprintf(`"%s" -service_run`, winvncExe), "/f"}
		executor.RunSilent("reg", regArgs...)

		// Inicia o processo Tray Icon na sessão do usuário atual
		executor.RunAsync(winvncExe, "-service_run")

		logger.LogSuccess("UltraVNC configurado, Serviço operando e Ícone ativado.")
		logger.WriteLog("UltraVNC instalado via INF, INI injetado e TrayIcon ativado.", "SUCCESS", logger.Green)
		report.AddDeployReport("UltraVNC", "Provisionamento", report.StatusSucesso, "Instalado, Serviço nativo e Ícone OK")
	}
}

// Função auxiliar simples para a injeção do .ini
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
