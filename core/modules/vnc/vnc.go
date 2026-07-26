package vnc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

type Installer struct {
	TempDir       string
	VncExePath    string
	VncIniPath    string
	VncInfPath    string
	InstallPath   string
	WinvncExe     string
	Uninstaller   string
	BaseVncFolder string
}

func New(tempDir string) *Installer {
	baseVncFolder := filepath.Join(os.Getenv("ProgramFiles"), "uvnc bvba")
	installPath := filepath.Join(baseVncFolder, "UltraVNC")

	return &Installer{
		TempDir:       tempDir,
		VncExePath:    filepath.Join(tempDir, "UltraVNC", "UltraVNC_Setup.exe"),
		VncIniPath:    filepath.Join(tempDir, "UltraVNC", "ultravnc.ini"),
		VncInfPath:    filepath.Join(tempDir, "UltraVNC", "ultravnc.inf"),
		InstallPath:   installPath,
		WinvncExe:     filepath.Join(installPath, "winvnc.exe"),
		Uninstaller:   filepath.Join(installPath, "unins000.exe"),
		BaseVncFolder: baseVncFolder,
	}
}

// Verifica se o UltraVNC já está instalado
func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.WinvncExe)
	return err == nil
}

// Desinstala o UltraVNC completamente, incluindo serviço, registro e arquivos residuais
func (i *Installer) Uninstall() {
	logger.LogStep("Executando Clean Nuke do UltraVNC...")

	// 1. Mata o processo do Tray Icon para liberar o executável
	executor.RunSilent("taskkill", "/F", "/IM", "winvnc.exe")
	time.Sleep(1 * time.Second)

	// 2. Remove a persistência do ícone no Registro customizado
	executor.RunSilent("reg", "delete", `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "/v", "UltraVNC", "/f")

	// 3. Executa o desinstalador oficial silenciosamente
	if _, err := os.Stat(i.Uninstaller); err == nil {
		logger.WriteLog("Invocando desinstalador oficial...", "INFO", logger.Cyan)
		executor.RunSilent(i.Uninstaller, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")
		time.Sleep(2 * time.Second)
	}

	// 4. Erradica a pasta (Lida com o ultravnc.ini órfão que bloqueia exclusão nativa)
	if _, err := os.Stat(i.BaseVncFolder); err == nil {
		logger.WriteLog("Removendo ultravnc.ini e pastas residuais...", "INFO", logger.Cyan)
		os.RemoveAll(i.BaseVncFolder)
	}

	logger.LogSuccess("UltraVNC removido completamente do sistema.")
	report.AddDeployReport("UltraVNC", "Desinstalação", report.StatusSucesso, "Clean Nuke executado")
}

// Faz a checagem de segurança do conteúdo do arquivo baixado
func (i *Installer) ValidateINI() bool {
	file, err := os.Open(i.VncIniPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Lê apenas o 1º KB para máxima eficiência de memória (equivalente ao Get-Content -TotalCount 15)
	buffer := make([]byte, 1024)
	n, _ := file.Read(buffer)
	content := strings.ToLower(string(buffer[:n]))

	return strings.Contains(content, "[admin]") || strings.Contains(content, "passwd")
}

func (i *Installer) Download(wg *sync.WaitGroup) {
	defer wg.Done()
	logger.WriteLog("Iniciando download do UltraVNC e dependências...", "INFO", logger.Cyan)

	if i.IsInstalled() {
		logger.LogWarning("UltraVNC já está instalado. Pulando fase de rede.")
		return
	}

	vncTempDir := filepath.Join(i.TempDir, "UltraVNC")
	os.MkdirAll(vncTempDir, os.ModePerm)

	fila := []downloader.DownloadItem{
		{URL: vncUrl, Destination: i.VncExePath, Label: "Instalador UltraVNC", ExpectedMB: 1.0, MagicType: "EXE"},
		{URL: iniUrl, Destination: i.VncIniPath, Label: "Arquivo de Conf (ultravnc.ini)", ExpectedMB: 0, MagicType: "Nenhum"},
		{URL: infUrl, Destination: i.VncInfPath, Label: "Arquivo de Resposta (ultravnc.inf)", ExpectedMB: 0, MagicType: "Nenhum"},
	}

	if err := downloader.DownloadsParalelos(fila); err != nil {
		logger.WriteLog("Falha no download dos recursos do VNC.", "ERROR", logger.Red)
	}

	// Validação de integridade do arquivo de configuração recém-baixado
	if i.ValidateINI() {
		logger.LogSuccess("Conteúdo do ultravnc.ini validado com sucesso.")
	} else {
		logger.LogWarning("O arquivo ultravnc.ini baixado não possui a estrutura padrão do VNC.")
	}
}

func (i *Installer) Install() {
	logger.LogStep("Iniciando Módulo: UltraVNC Installer (Ponto Zero)")

	if i.IsInstalled() {
		logger.LogWarning("O UltraVNC já está instalado nesta máquina.")
		report.AddDeployReport("UltraVNC", "Instalação", report.StatusAviso, "Ignorado (Já estava instalado)")
		return
	}

	if _, err := os.Stat(i.VncExePath); os.IsNotExist(err) {
		logger.WriteLog("Instalador do VNC não encontrado. Abortando instalação.", "ERROR", logger.Red)
		report.AddDeployReport("UltraVNC", "Instalação", report.StatusFalha, "Download não concluído")
		return
	}

	logger.LogStep("Preparando ambiente e executando instalação silenciosa...")
	os.MkdirAll(i.InstallPath, os.ModePerm)
	copyFile(i.VncIniPath, filepath.Join(i.InstallPath, "ultravnc.ini"))
	logger.LogStep("Arquivo ultravnc.ini pré-injetado com sucesso.")

	installArgs := []string{
		"/VERYSILENT",
		fmt.Sprintf(`/LOADINF=%s`, i.VncInfPath),
		`/TASKS=installservice`,
		"/NORESTART",
	}

	exitCode, err := executor.RunSilent(i.VncExePath, installArgs...)
	if err != nil && exitCode != 0 && exitCode != 3010 {
		logger.LogWarning(fmt.Sprintf("Instalador retornou erro: %d", exitCode))
		logger.WriteLog("Falha na instalação do UltraVNC.", "ERROR", logger.Red)
		report.AddDeployReport("UltraVNC", "Instalação", report.StatusFalha, fmt.Sprintf("ExitCode %d", exitCode))
		return
	}

	logger.LogSuccess("Instalação base concluída e serviço registrado pelo desinstalador.")
	logger.LogStep("Ativando serviço e persistência do ícone...")

	if i.IsInstalled() {
		// Inicia o serviço do UltraVNC e adiciona a persistência do ícone no registro
		executor.RunSilent("sc", "start", "uvnc_service")

		regArgs := []string{"add", `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "/v", "UltraVNC", "/t", "REG_SZ", "/d", fmt.Sprintf(`"%s" -service_run`, i.WinvncExe), "/f"}
		executor.RunSilent("reg", regArgs...)

		// Limpa instâncias residuais na sessão do usuário antes de disparar a nova para evitar mensagens de conflito
		executor.RunSilent("taskkill", "/F", "/IM", "winvnc.exe")
		time.Sleep(1 * time.Second)

		executor.RunAsync(i.WinvncExe, "-service_run")

		logger.LogSuccess("UltraVNC configurado, Serviço operando e Ícone ativado.")
		logger.WriteLog("UltraVNC instalado via INF, INI injetado e TrayIcon ativado.", "SUCCESS", logger.Green)
		report.AddDeployReport("UltraVNC", "Provisionamento", report.StatusSucesso, "Instalado, Serviço nativo e Ícone OK")
	}

	// Limpeza dos instaladores para o Zero Touch
	os.RemoveAll(filepath.Join(i.TempDir, "UltraVNC"))
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
