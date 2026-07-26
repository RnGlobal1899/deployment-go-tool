package winrar

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

type Installer struct {
	FallbackPath string
}

func New(tempDir string) *Installer {
	return &Installer{
		FallbackPath: filepath.Join(tempDir, "WinRAR_Fallback.exe"),
	}
}

func (i *Installer) Download(wg *sync.WaitGroup) {
	defer wg.Done()
	logger.LogStep("Baixando Fallback corporativo do WinRAR (Dropbox)...")
	fila := []downloader.DownloadItem{
		{
			URL:         "https://www.dropbox.com/scl/fi/l5kns7uhdk0bay9edzvs4/winrar-x64-722br.exe?rlkey=tu11ul2ozibefye8sj6r21cyr&st=vc67ly6x&dl=1",
			Destination: i.FallbackPath,
			Label:       "WinRAR Fallback",
			ExpectedMB:  3.0,
			MagicType:   "EXE",
		},
	}

	err := downloader.DownloadsParalelos(fila)

	if err != nil {
		logger.WriteLog(fmt.Sprintf("Falha no download do fallback do WinRAR: %v", err), "ERROR", logger.Red)
	}
}

func (i *Installer) Install() {
	logger.LogStep("Instalando RARLab.WinRAR (Tentativa 1: Winget)...")

	exitCode, err := executor.RunSilent("winget.exe", "install", "--id", "RARLab.WinRAR", "--exact", "--silent", "--accept-package-agreements", "--accept-source-agreements")
	if err == nil && exitCode == 0 {
		logger.LogSuccess("WinRAR instalado com sucesso via Winget.")
		report.AddDeployReport("Utilitários", "RARLab.WinRAR", "Sucesso", "Instalado via Winget")
		os.Remove(i.FallbackPath)
		return
	}

	logger.LogWarning(fmt.Sprintf("Winget falhou (Exit: %d). Acionando Fallback corporativo...", exitCode))

	if _, err := os.Stat(i.FallbackPath); os.IsNotExist(err) {
		logger.WriteLog("Executável de Fallback do WinRAR não encontrado.", "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "RARLab.WinRAR", "Falha", "Winget falhou e Fallback ausente")
		return
	}

	exitCodeFb, errFb := executor.RunSilent(i.FallbackPath, "/S")
	if errFb != nil || (exitCodeFb != 0 && exitCodeFb != 3010) {
		logger.WriteLog(fmt.Sprintf("Falha crítica na instalação via Fallback. ExitCode: %d", exitCodeFb), "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "RARLab.WinRAR", "Falha", fmt.Sprintf("Fallback abortou (ExitCode %d)", exitCodeFb))
	} else {
		logger.LogSuccess("WinRAR instalado com sucesso via Fallback.")
		report.AddDeployReport("Utilitários", "RARLab.WinRAR", "Sucesso", "Instalado via Fallback (Dropbox)")
	}

	os.Remove(i.FallbackPath)
}
