package wps

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"grc-deploy/core/downloader"
	"grc-deploy/core/executor"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"
)

// Erro utilizado para interromper o filepath.Walk quando o arquivo desejado for encontrado
var ErrFound = errors.New("arquivo encontrado")

// Estrutura os métodos exportados para o frontend
type WpsManager struct{}

func NewWpsManager() *WpsManager {
	return &WpsManager{}
}

// Emite um alerta lógico no caso a instalação esteja no perfil Admin
func (w *WpsManager) CheckAdminProfile() {
	usr, err := user.Current()
	if err == nil {
		username := strings.ToLower(usr.Username)
		logger.LogStep(fmt.Sprintf("Usuário atual detectado: %s", usr.Username))

		if strings.Contains(username, "administrator") || strings.Contains(username, "administrador") {
			logger.LogWarning("Aviso: O WPS Office costuma gravar dados no AppData do perfil logado.")
			logger.LogWarning(fmt.Sprintf("Executando como '%s'. A instalação prosseguirá na conta administrativa.", usr.Username))
		}
	}
}

// Verifica se o WPS Office já está instalado no sistema
func (w *WpsManager) CheckInstalled() (bool, string) {
	progFiles86 := os.Getenv("ProgramFiles(x86)")
	localAppData := os.Getenv("LOCALAPPDATA")

	// Base de diretórios padrão do Kingsoft/WPS
	bases := []string{
		filepath.Join(progFiles86, "Kingsoft"),
		filepath.Join(localAppData, "Kingsoft"),
	}

	var uninstPath string

	for _, base := range bases {
		// Verifica se existe antes de varrer
		if _, err := os.Stat(base); os.IsNotExist(err) {
			continue
		}

		// Busca recursiva otimizada
		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.ToLower(d.Name()) == "uninst.exe" {
				uninstPath = path
				return ErrFound
			}
			return nil
		})

		if uninstPath != "" {
			break
		}
	}

	return uninstPath != "", uninstPath
}

// Aaciona o desinstalador Win32 com supressão total de UI
func (w *WpsManager) Uninstall(uninstallerPath string) error {
	logger.LogStep("Iniciando desinstalação silenciosa do WPS Office...")

	exitCode, err := executor.RunSilent(uninstallerPath, "/S")
	if err != nil {
		report.AddDeployReport("WPS Office", "Desinstalação", "Falha", err.Error())
		logger.WriteLog(fmt.Sprintf("Erro ao desinstalar WPS: %v", err), "ERROR", logger.Red)
		return fmt.Errorf("falha crítica na desinstalação: %v", err)
	}

	report.AddDeployReport("WPS Office", "Desinstalação", "Sucesso", "")
	logger.LogSuccess(fmt.Sprintf("WPS Office desinstalado com sucesso (ExitCode: %d).", exitCode))
	return nil
}

// Orquestra toda a esteira do WPS (Fase de Rede + Fase de Instalação Sequencial)
// Se 'forceReinstall' for false, ele ignora caso já esteja instalado.
func (w *WpsManager) Deploy(forceReinstall bool) error {
	logger.LogStep("Módulo WPS Office iniciado pelo Master Deploy.")
	w.CheckAdminProfile()

	// Validação Pré-Deploy
	isInstalled, uninstPath := w.CheckInstalled()
	if isInstalled {
		if !forceReinstall {
			logger.LogWarning("WPS Office já está instalado nesta máquina. Deploy ignorado pelo modo autônomo.")
			report.AddDeployReport("WPS Office", "Instalação", "Aviso", "Ignorado (Já estava instalado)")
			return nil
		}

		logger.LogStep("Forçando reinstalação. Desinstalando versão atual...")
		if err := w.Uninstall(uninstPath); err != nil {
			return err
		}
	}

	// Fase 1: Rede (Assíncrona)
	tempDir := "C:\\TI_Setup_Temp\\WPSOffice"
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return fmt.Errorf("falha ao criar diretório temporário: %v", err)
	}

	installerDest := filepath.Join(tempDir, "WPSOffice_Setup.exe")
	downloadURL := "https://www.dropbox.com/scl/fi/dqfmup7grrfsx613nbvb4/WPSOffice_11.2.0.11074-1.exe?rlkey=745qmxduzvyj8cu68nephzmxg&st=lkkeqy22&dl=1"

	logger.LogStep("Iniciando download do instalador do WPS Office...")

	var fila []downloader.DownloadItem

	fila = append(fila, downloader.DownloadItem{
		URL:         downloadURL,
		Destination: installerDest,
		Label:       "WPS Office",
		ExpectedMB:  150.0,
		MagicType:   "EXE",
	})

	err := downloader.DownloadsParalelos(fila)

	if err != nil {
		logger.WriteLog(fmt.Errorf("Falha no download do WPS Office: %w", err).Error(), "ERROR", logger.Red)
		report.AddDeployReport("WPS Office", "Download", "FALHA", fmt.Sprintf("Erro: %v", err))
	}

	// Fase 2: Instalação (Sequencial)
	logger.LogStep("Iniciando instalação silenciosa do WPS Office. Aguarde...")
	exitCode, err := executor.RunSilent(installerDest, "/S")
	if err != nil {
		report.AddDeployReport("WPS Office", "Instalação", "Falha", err.Error())
		logger.WriteLog(fmt.Sprintf("Falha ao executar instalador (ExitCode: %d): %v", exitCode, err), "ERROR", logger.Red)
		return err
	}

	// Fase 3: Pós-Validação (Verificação Dupla de Sucesso)
	logger.LogStep("Realizando verificação dupla pós-instalação...")
	successCheck, _ := w.CheckInstalled()
	if !successCheck {
		msg := "Processo de instalação finalizou, porém as pastas/uninst.exe do Kingsoft não foram detectadas."
		report.AddDeployReport("WPS Office", "Instalação", "Falha", "Diretório não detectado pós-instalação")
		logger.WriteLog(msg, "WARNING", logger.Yellow)
		return errors.New("falha na validação pós-instalação")
	}

	report.AddDeployReport("WPS Office", "Instalação", "Sucesso", "")
	logger.LogSuccess(fmt.Sprintf("WPS Office instalado com sucesso (ExitCode: %d).", exitCode))

	return nil
}
