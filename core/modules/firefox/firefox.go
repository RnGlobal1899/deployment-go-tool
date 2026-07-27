package firefox

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
	FilePath    string
	InstallPath string
}

func New(tempDir string) *Installer {
	return &Installer{
		FilePath:    filepath.Join(tempDir, "Firefox_Installer.exe"),
		InstallPath: filepath.Join(os.Getenv("ProgramFiles"), "Mozilla Firefox", "firefox.exe"),
	}
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.InstallPath)
	return err == nil
}

func (i *Installer) Download(wg *sync.WaitGroup) {
	defer wg.Done()
	logger.LogStep("Baixando Mozilla Firefox (URL Dinâmica Oficial)...")
	fila := []downloader.DownloadItem{
		{
			URL:         "https://download.mozilla.org/?product=firefox-latest-ssl&os=win64&lang=pt-BR",
			Destination: i.FilePath,
			Label:       "Mozilla Firefox",
			ExpectedMB:  50.0,
			MagicType:   "EXE",
		},
	}

	err := downloader.DownloadsParalelos(fila)

	if err != nil {
		logger.WriteLog(fmt.Sprintf("Falha no download do Firefox: %v", err), "ERROR", logger.Red)
	}
}

func (i *Installer) Install() {
	logger.LogStep("Instalando Mozilla Firefox...")
	if _, err := os.Stat(i.FilePath); os.IsNotExist(err) {
		logger.WriteLog("Executável do Firefox não encontrado. Abortando.", "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "Mozilla Firefox", "Falha", "Download não concluído")
		return
	}

	exitCode, err := executor.RunSilent(i.FilePath, "/S")
	if err != nil || (exitCode != 0 && exitCode != 3010) {
		logger.WriteLog(fmt.Sprintf("Falha na instalação do Firefox. ExitCode: %d", exitCode), "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "Mozilla Firefox", "Falha", fmt.Sprintf("ExitCode %d", exitCode))
	} else {
		logger.LogSuccess("Mozilla Firefox instalado com sucesso.")
		report.AddDeployReport("Utilitários", "Mozilla Firefox", "Sucesso", "Instalado via URL Oficial")
	}

	os.Remove(i.FilePath)
}
