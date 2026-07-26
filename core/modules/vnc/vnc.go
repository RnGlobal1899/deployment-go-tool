package vnc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

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
	TempDir     string
	VncExePath  string
	VncIniPath  string
	VncInfPath  string
	InstallPath string
	WinvncExe   string
}

func New(tempDir string) *Installer {
	installPath := filepath.Join(os.Getenv("ProgramFiles"), "uvnc bvba", "UltraVNC")
	return &Installer{
		TempDir:     tempDir,
		VncExePath:  filepath.Join(tempDir, "UltraVNC", "UltraVNC_Setup.exe"),
		VncIniPath:  filepath.Join(tempDir, "UltraVNC", "ultravnc.ini"),
		VncInfPath:  filepath.Join(tempDir, "UltraVNC", "ultravnc.inf"),
		InstallPath: installPath,
		WinvncExe:   filepath.Join(installPath, "winvnc.exe"),
	}
}

func (i *Installer) Download(wg *sync.WaitGroup) {
	defer wg.Done()
	logger.WriteLog("Iniciando download do UltraVNC e dependências...", "INFO", logger.Cyan)

	if _, err := os.Stat(i.WinvncExe); err == nil {
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

	// Executa os downloads em paralelo e registra falhas, mas não interrompe o fluxo
	if err := downloader.DownloadsParalelos(fila); err != nil {
		logger.WriteLog("Falha no download dos recursos do VNC.", "ERROR", logger.Red)
	}
}

func (i *Installer) Install() {
	logger.LogStep("Iniciando Módulo: UltraVNC Installer (Ponto Zero)")

	if _, err := os.Stat(i.WinvncExe); err == nil {
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

	if _, err := os.Stat(i.WinvncExe); err == nil {
		executor.RunSilent("sc", "start", "uvnc_service")

		regArgs := []string{"add", `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, "/v", "UltraVNC", "/t", "REG_SZ", "/d", fmt.Sprintf(`"%s" -service_run`, i.WinvncExe), "/f"}
		executor.RunSilent("reg", regArgs...)

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
