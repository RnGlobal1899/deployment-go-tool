package anydesk

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
	FilePath string
}

func New(tempDir string) *Installer {
	return &Installer{
		FilePath: filepath.Join(tempDir, "AnyDesk_Installer.exe"),
	}
}

func (i *Installer) Download(wg *sync.WaitGroup) {
	defer wg.Done()
	logger.LogStep("Baixando AnyDesk (URL Dinâmica Oficial)...")
	fila := []downloader.DownloadItem{
		{
			URL:         "https://download.anydesk.com/AnyDesk.exe",
			Destination: i.FilePath,
			Label:       "AnyDesk",
			ExpectedMB:  5.0,
			MagicType:   "EXE",
		},
	}

	err := downloader.DownloadsParalelos(fila)

	if err != nil {
		logger.WriteLog(fmt.Sprintf("Falha no download do AnyDesk: %v", err), "ERROR", logger.Red)
	}
}

func (i *Installer) Install() {
	logger.LogStep("Instalando AnyDesk...")
	if _, err := os.Stat(i.FilePath); os.IsNotExist(err) {
		logger.WriteLog("Executável do AnyDesk não encontrado. Abortando.", "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "AnyDesk", "Falha", "Download não concluído")
		return
	}

	// Abstração nativa do Váriavel de Ambiente no Go
	pf := os.Getenv("ProgramFiles(x86)")
	if pf == "" {
		pf = `C:\Program Files (x86)`
	}
	installDir := filepath.Join(pf, "AnyDesk")

	exitCode, err := executor.RunSilent(i.FilePath, "--install", installDir, "--start-with-win", "--silent")
	if err != nil || (exitCode != 0 && exitCode != 3010) {
		logger.WriteLog(fmt.Sprintf("Falha na instalação do AnyDesk. ExitCode: %d", exitCode), "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "AnyDesk", "Falha", fmt.Sprintf("ExitCode %d", exitCode))
	} else {
		logger.LogSuccess("AnyDesk instalado com sucesso.")
		report.AddDeployReport("Utilitários", "AnyDesk", "Sucesso", "Instalado via URL Oficial")
	}

	os.Remove(i.FilePath)
}
