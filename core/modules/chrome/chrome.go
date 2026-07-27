package chrome

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

// Struct para gerenciar a instalação do software
type Installer struct {
	FilePath    string
	InstallPath string
}

// Função para criar uma nova instância do instalador
func New(tempDir string) *Installer {
	return &Installer{
		FilePath:    filepath.Join(tempDir, "Chrome_Installer.exe"),
		InstallPath: filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
	}
}

// Verifica se o Google Chrome já está instalado
func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.InstallPath)
	return err == nil
}

// Baixa o instalador do Google Chrome
func (i *Installer) Download(wg *sync.WaitGroup) {
	defer wg.Done()
	logger.LogStep("Baixando Google Chrome (URL Dinâmica Oficial)...")
	fila := []downloader.DownloadItem{
		{
			URL:         "https://dl.google.com/chrome/install/latest/chrome_installer.exe",
			Destination: i.FilePath,
			Label:       "Google Chrome",
			ExpectedMB:  5.0,
			MagicType:   "EXE",
		},
	}

	err := downloader.DownloadsParalelos(fila)

	if err != nil {
		logger.WriteLog(fmt.Sprintf("Falha no download do Chrome: %v", err), "ERROR", logger.Red)
	}
}

// Instala o Google Chrome silenciosamente
func (i *Installer) Install() {
	logger.LogStep("Instalando Google Chrome...")
	if _, err := os.Stat(i.FilePath); os.IsNotExist(err) {
		logger.WriteLog("Executável do Chrome não encontrado. Abortando.", "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "Google Chrome", "Falha", "Download não concluído")
		return
	}

	exitCode, err := executor.RunSilent(i.FilePath, "/silent", "/install")
	if err != nil || (exitCode != 0 && exitCode != 3010) {
		logger.WriteLog(fmt.Sprintf("Falha na instalação do Chrome. ExitCode: %d", exitCode), "ERROR", logger.Red)
		report.AddDeployReport("Utilitários", "Google Chrome", "Falha", fmt.Sprintf("ExitCode %d", exitCode))
	} else {
		logger.LogSuccess("Google Chrome instalado com sucesso.")
		report.AddDeployReport("Utilitários", "Google Chrome", "Sucesso", "Instalado via URL Oficial")
	}

	// Remove o instalador após a instalação
	os.Remove(i.FilePath)
}
