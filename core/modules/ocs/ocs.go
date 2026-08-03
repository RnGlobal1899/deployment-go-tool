package ocs

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"grc-deploy/core/downloader"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"
)

// Estrutura os métodos exportados para o frontend via Wails
type OcsManager struct{}

func NewOcsManager() *OcsManager {
	return &OcsManager{}
}

// Verifica a existência do OCS Agent no diretório padrão do sistema.
func (o *OcsManager) CheckInstalled() bool {
	installPath := filepath.Join(os.Getenv("ProgramFiles(x86)"), "OCS Inventory Agent")
	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		return true
	}
	return false
}

// Oorquestra a esteira do OCS: Fase de Rede (Assíncrona) e Instalação (Interativa com GUI).
func (o *OcsManager) Deploy(forceReinstall bool) error {
	logger.LogStep("Módulo OCS Agent iniciado pelo Master Deploy.")

	// Validação Pré-Deploy
	if o.CheckInstalled() {
		if !forceReinstall {
			logger.LogSuccess("OCS Inventory Agent já está instalado na máquina.")
			logger.WriteLog("OCS Agent detectado. Instalação ignorada.", "INFO", logger.Green)
			report.AddDeployReport("OCS Agent", "Instalação", "Aviso", "Ignorado (Já estava instalado)")
			return nil
		}
		logger.LogWarning("Aviso: Reinstalação forçada do OCS iniciada (instalação será feita por cima da versão atual).")
	}

	// Fase 1: Rede (Assíncrona)
	tempDir := "C:\\TI_Setup_Temp\\OCS"
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return fmt.Errorf("falha ao criar diretório temporário: %v", err)
	}

	installerDest := filepath.Join(tempDir, "OCS-NG-Windows-Agent-Setup.exe")
	downloadURL := "https://www.dropbox.com/scl/fi/118md17xwp9nvregig09w/OCS-NG-Windows-Agent-Setup.exe?rlkey=cfv2836xk3p743gui0b8jh30b&st=95sr99nx&dl=1"

	var fila []downloader.DownloadItem
	fila = append(fila, downloader.DownloadItem{
		URL:         downloadURL,
		Destination: installerDest,
		Label:       "Instalador OCS Agent",
		ExpectedMB:  3.0,
		MagicType:   "EXE",
	})

	err := downloader.DownloadsParalelos(fila)
	if err != nil {
		logger.WriteLog(fmt.Errorf("Falha no download do OCS Agent: %w", err).Error(), "ERROR", logger.Red)
		report.AddDeployReport("OCS Agent", "Download", "FALHA", fmt.Sprintf("Erro: %v", err))
		return err
	}

	// Fase 2: Instalação (Manual / Interativa)
	fmt.Println("\n  ============================================================")
	fmt.Println("  [ATENÇÃO] CONFIGURAÇÃO MANUAL DO OCS REQUERIDA")
	fmt.Println("  ============================================================")
	fmt.Println("  O instalador será aberto. Preencha os dados conforme abaixo:")
	fmt.Println("")
	fmt.Printf("  ▶ Server URL:      http://10.99.1.25/ocsinventory\n")
	fmt.Printf("  ▶ Validate cert.:  Deixe o padrão cacert.pem\n")
	fmt.Printf("  ▶ Proxy/Outros:    Apenas avance (Next)\n")
	fmt.Printf("  ▶ Marcar:          1º, 3º e 6º opção\n")
	fmt.Printf("  ▶ TAG:             Setor - PT: Patrimônio da máquina\n")
	fmt.Println("  ============================================================")
	fmt.Println("")
	logger.LogStep("Aguardando o usuário concluir a instalação na janela do OCS...")

	// Executa o OCS Agent Setup e aguarda o usuário finalizar a instalação manualmente
	cmd := exec.Command(installerDest)
	err = cmd.Run() // Trava a goroutine atual até que a janela do instalador seja fechada

	exitCode := 0
	if err != nil {
		// Tenta extrair o ExitCode caso o instalador retorne algo diferente de 0 (ex: cancelamento)
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			report.AddDeployReport("OCS Agent", "Instalação", "Falha", "Erro ao abrir o executável")
			logger.WriteLog(fmt.Sprintf("Erro OCS: %v", err), "ERROR", logger.Red)
			return err
		}
	}

	// Fase 3: Pós-Validação (Verificação de Sucesso)
	logger.LogStep("Realizando verificação pós-instalação...")
	if o.CheckInstalled() {
		report.AddDeployReport("OCS Agent", "Instalação", "Sucesso", "Instalação manual concluída")
		logger.LogSuccess(fmt.Sprintf("Instalação do OCS concluída pelo usuário (ExitCode: %d).", exitCode))
		logger.WriteLog("OCS Agent instalado com sucesso.", "SUCCESS", logger.Green)
	} else {
		msg := "Instalador finalizado, mas a pasta do OCS não foi encontrada."
		report.AddDeployReport("OCS Agent", "Instalação", "Falha", "Cancelado ou falha no instalador")
		logger.WriteLog(msg, "WARNING", logger.Yellow)
		return errors.New("falha na validação pós-instalação")
	}

	return nil
}
