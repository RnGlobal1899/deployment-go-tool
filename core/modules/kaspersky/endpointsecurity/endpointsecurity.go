package endpointsecurity

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"grc-deploy/core/downloader"
	"grc-deploy/core/executor"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"
)

const (
	EndpointURL       = "https://www.dropbox.com/scl/fi/66qhf9ffyxxzyb5e3ym9y/KES_11.7.0.669.zip?rlkey=decyrsu9jaz1hl1sofbg7ehh6&st=sqpw9bv9&dl=1"
	InstallDir        = "C:\\Program Files (x86)\\Kaspersky Lab\\Kaspersky Endpoint Security for Windows"
	TempFolder        = "C:\\TI_Setup_Temp\\Kaspersky"
	EndpointZip       = "C:\\TI_Setup_Temp\\Kaspersky\\endpoint.zip"
	EndpointExtracted = "C:\\TI_Setup_Temp\\Kaspersky\\EndpointFiles"
)

// Orquestra o download de forma totalmente assíncrona.
func DownloadAsync(wg *sync.WaitGroup) {
	defer wg.Done()
	if CheckCached() {
		logger.LogStep("Endpoint.zip / Setup pré-extraído validado no cache. Omitindo banda de rede.")
		report.AddDeployReport("Endpoint Security", "Download", "SUCESSO", "Instalador já em cache.")
		return
	}

	var fila []downloader.DownloadItem

	os.MkdirAll(TempFolder, os.ModePerm)
	fila = append(fila, downloader.DownloadItem{
		URL:         EndpointURL,
		Destination: EndpointZip,
		Label:       "Endpoint Security",
		ExpectedMB:  150.0,
		MagicType:   "ZIP",
	})

	err := downloader.DownloadsParalelos(fila)

	if err != nil {
		logger.WriteLog(fmt.Errorf("Falha no download do Endpoint Security: %w", err).Error(), "ERROR", logger.Red)
		report.AddDeployReport("Endpoint Security", "Download", "FALHA", fmt.Sprintf("Erro: %v", err))
	}
}

// Confere integridade do ZIP e validação da extração prévia se existir.
func CheckCached() bool {
	info, err := os.Stat(filepath.Join(EndpointExtracted, "setup.exe"))
	if err == nil && info.Size() > 1000000 {
		return true
	}
	infoZip, errZip := os.Stat(EndpointZip)
	return errZip == nil && infoZip.Size() > 50000000
}

// Mapeia o executável de proteção principal avp.exe.
func CheckInstalled() bool {
	_, err := os.Stat(filepath.Join(InstallDir, "avp.exe"))
	return !os.IsNotExist(err)
}

// Bloqueia redownloads caso o AVP já esteja ativo. Aplica licença faltante sob demanda.
func InstallOrActivate(activationCode string) error {
	if CheckInstalled() {
		logger.LogStep("Kaspersky Endpoint já instalado. Acionamento da instalação de binários ignorada.")
		report.AddDeployReport("Endpoint Security", "Install", "SUCESSO", "Endpoint já instalado.")
		if activationCode != "" {
			return ActivateLicense(activationCode)
		}
		return nil
	}
	return Install(activationCode)
}

// Assume a Extração Segura contra Path Traversal e injeção massiva via setup.exe.
func Install(activationCode string) error {
	logger.LogStep("Orquestrando a instalação silenciosa do Kaspersky Endpoint Security (pode demorar vários minutos)...")

	// Função auxiliar para procurar o setup.exe independente da subpasta que o ZIP gerar
	findSetupExe := func() string {
		var pathEncontrado string
		filepath.Walk(EndpointExtracted, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && strings.ToLower(info.Name()) == "setup.exe" {
				pathEncontrado = path
				return filepath.SkipDir // Para a busca assim que encontrar
			}
			return nil
		})
		return pathEncontrado
	}

	setupPath := findSetupExe()

	if setupPath == "" {
		logger.LogStep("Acionando rotina de extração segura...")
		errExt := extractZip(EndpointZip, EndpointExtracted)
		if errExt != nil {
			report.AddDeployReport("Endpoint Security", "Extração", "FALHA", errExt.Error())
			return fmt.Errorf("engine de extração reportou erro: %v", errExt)
		}

		// Procura novamente após extrair
		setupPath = findSetupExe()
		if setupPath == "" {
			return fmt.Errorf("setup.exe não encontrado no pacote extraído")
		}
	}

	setupDir := filepath.Dir(setupPath)
	logger.LogStep("Executando setup.exe silenciosamente...")
	logger.WriteLog(fmt.Sprintf("Diretório de trabalho (WorkingDir): %s", setupDir), "INFO", logger.Cyan)

	// Flag /pALLOWREBOOT=0 injetada para que a esteira continue intacta. Supressão de UI via /s.
	psInstallCmd := fmt.Sprintf(`
	$proc = Start-Process -FilePath "%s" -ArgumentList "/s /pEULA=1 /pPRIVACYPOLICY=1 /pALLOWREBOOT=0" -WorkingDirectory "%s" -Wait -NoNewWindow -PassThru
	exit $proc.ExitCode
	`, setupPath, setupDir)

	exitCode, err := executor.RunSilent("powershell.exe", "-NoProfile", "-Command", psInstallCmd)

	if err != nil || (exitCode != 0 && exitCode != 3010) {
		report.AddDeployReport("Endpoint Security", "Instalação", "FALHA", fmt.Sprintf("ExitCode: %d", exitCode))
		return fmt.Errorf("o processo setup.exe sofreu crash: %v (ExitCode: %d)", err, exitCode)
	}

	report.AddDeployReport("Endpoint Security", "Instalação", "Sucesso", "Binário core AVP inserido no disco.")

	if activationCode != "" {
		return ActivateLicense(activationCode)
	}
	return nil
}

// Utiliza chamadas restritas ao CLI de segurança da Kaspersky (avp.com).
func ActivateLicense(code string) error {
	avpCom := filepath.Join(InstallDir, "avp.com")
	if _, err := os.Stat(avpCom); os.IsNotExist(err) {
		report.AddDeployReport("Endpoint Security", "Ativação", "FALHA", "avp.com não encontrado.")
		return fmt.Errorf("binário avp.com de controle ausente, impossível acionar licença")
	}

	_, err := executor.RunSilent(avpCom, "LICENSE", "/add", code)
	if err != nil {
		report.AddDeployReport("Endpoint Security", "Ativação", "Aviso", "Falha na validação com KSC ou chave revogada.")
		return fmt.Errorf("falha ao ativar chave comercial: %v", err)
	}

	report.AddDeployReport("Endpoint Security", "Ativação", "Sucesso", "Licença injetada ativamente na suíte (avp.com).")
	return nil
}

// Retornado quando o parser identifica bloqueio por credenciais
var ErrPasswordRequired = errors.New("KLADMIN_AUTH_REQUIRED")

// Constrói um script dinâmico localizando raízes do sistema via WMI/PowerShell, forçando o Drop
// Inclui injeção de KLLOGIN e KLPASSWD de forma encapsulada prevenindo ExitCode 3 por KLAdmin.
func Uninstall(klLogin, klPasswd string) error {
	logger.LogStep("Iniciando rotina cirúrgica de Clean Nuke do Endpoint Security...")

	// Define o arquivo de log para o MSI reportar detalhes da operação
	msiLogPath := filepath.Join(TempFolder, "kes_uninstall.log")

	// Limpa o log anterior para evitar falsos positivos na leitura
	os.Remove(msiLogPath)

	// Se as credenciais KLAdmin forem fornecidas, são injetadas no comando de desinstalação
	if klLogin != "" && klPasswd != "" {
		logger.LogStep("Tentativa de desinstalação com credenciais KLAdmin ativada.")
	} else {
		logger.LogStep("Tentativa 1: Desinstalação silenciosa sem credenciais.")
	}

	psCommand := fmt.Sprintf(`
	$app = Get-ItemProperty -Path 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*', 'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*' -ErrorAction SilentlyContinue | Where-Object { $_.DisplayName -like '*Kaspersky Endpoint Security*' } | Select-Object -First 1
	if ($app) {
		$guid = $app.PSChildName
		$logPath = "%s"
		$argsList = "/x `+"`\"$guid`\""+` /qn /norestart /L*v `+"`\"$logPath`\""+`"
		
		if ("%s" -ne "" -and "%s" -ne "") {
			$argsList = "/x `+"`\"$guid`\""+` KLLOGIN=`+"`\"%s`\""+` KLPASSWD=`+"`\"%s`\""+` /qn /norestart /L*v `+"`\"$logPath`\""+`"
		}
		
		$proc = Start-Process "msiexec.exe" -ArgumentList $argsList -Wait -NoNewWindow -PassThru
		exit $proc.ExitCode
	} else {
		exit 1605
	}`, msiLogPath, klLogin, klPasswd, klLogin, klPasswd)

	// Recupera o ExitCode do PowerShell/MSI
	exitCode, err := executor.RunSilent("powershell.exe", "-NoProfile", "-Command", psCommand)

	// Avaliação do fluxo de sucesso/falha
	if err != nil || (exitCode != 0 && exitCode != 3010 && exitCode != 1605) { // 1605 = Já removido, 3010 = Reboot pendente
		logger.WriteLog(fmt.Sprintf("Processo msiexec retornou ExitCode atípico: %d", exitCode), "WARNING", logger.Yellow)

		// Se as credenciais não forem fornecidas, analisa o log para descobrir o motivo da falha
		if klLogin == "" && klPasswd == "" {
			if checkLogForPasswordReq(msiLogPath) {
				logger.WriteLog("Log indica proteção ativa. Credenciais KLAdmin são requeridas.", "INFO", logger.Cyan)
				return ErrPasswordRequired
			}
		}

		return fmt.Errorf("desinstalação falhou (ExitCode: %d). Verifique o log: %s", exitCode, msiLogPath)
	}

	report.AddDeployReport("Endpoint Security", "Desinstalação", "Sucesso", "Software removido com sucesso.")
	return nil
}

// Faz o parsing do arquivo de log do MSI usando regex procurando indicadores multilingues de bloqueio por senha.
func checkLogForPasswordReq(logPath string) bool {
	content, err := os.ReadFile(logPath)
	if err != nil {
		logger.WriteLog("Não foi possível ler o log do MSI para análise de falha.", "WARNING", logger.Yellow)
		return false
	}

	logData := string(content)

	// Mapeia variações em inglês, português e referências diretas às chaves KLAdmin
	// `(?i)` torna o regex case-insensitive
	pattern := `(?i)(password\sprotection\senabled|senha\sincorreta|KLPASSWD|KLAdminPasswd|proteção\spor\ssenha)`
	matched, _ := regexp.MatchString(pattern, logData)

	return matched
}

// --------------------------------- //
//         Funções internas          //
// --------------------------------- //

// Extrai o ZIP de forma segura, prevenindo Path Traversal e injeção de arquivos maliciosos.
func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Sanitiza o diretório de destino fora do loop para otimização
	destClean := filepath.Clean(dest)

	for _, f := range r.File {
		fpath := filepath.Join(destClean, f.Name)

		// SAFE ZipSlip: Garante que o arquivo não escape do diretório alvo
		// A condição (fpath != destClean) resolve o falso positivo de metadados de raiz do Dropbox
		if !strings.HasPrefix(fpath, destClean+string(os.PathSeparator)) && fpath != destClean {
			return fmt.Errorf("%s: caminho de arquivo ilegal", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
