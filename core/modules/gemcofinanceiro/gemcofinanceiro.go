package gemcofinanceiro

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"grc-deploy/core/downloader"
	"grc-deploy/core/executor"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"

	"golang.org/x/sys/windows/registry"
)

// Constantes fundamentais
const (
	BaseZipName = "Pacote_v11.zip"
	BaseURL     = "https://www.dropbox.com/scl/fo/gnaeldk7g7djq1rckxghj/AAGeecw0fMoLJmx-UNvsugU?rlkey=inb3tblr6zken8srwtcnq6pbz&st=mhoa5ao4&dl=1"
	BaseDir     = "C:\\Gemco\\GemcoFinanceiro"
	RegPath     = `SOFTWARE\WOW6432Node\Gemco\GemcoFinanceiro`
)

// Define a estrutura das atualizações do Financeiro
type UpdateCatalog struct {
	URL      string
	Type     string
	RegKey   string
	RegValue string
}

// Catálogo
var Catalog = map[string]UpdateCatalog{
	"GemcoFinanceiroSP3-00-1.exe":     {URL: "https://www.dropbox.com/scl/fi/yol4lw2kiyqrvhvl3g80w/GemcoFinanceiroSP3-00-1.exe?rlkey=49ggj346pze1vz453z9kl25vn&st=i9htqdo0&dl=1", Type: "SP", RegKey: "SP", RegValue: "SP3.00.1"},
	"GEMCOFINANCEIROSPCustom1-44.EXE": {URL: "https://www.dropbox.com/scl/fi/uuvm00caaha96fhxpixwa/GEMCOFINANCEIROSPCustom1-44.EXE?rlkey=sqlu10y1g3bycybph289boa4d&st=lqu8v8da&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c1.44"},
	"GEMCOFINANCEIROSPCustom1-46.EXE": {URL: "https://www.dropbox.com/scl/fi/5pvaxkiwx6mh7kyl6lctd/GEMCOFINANCEIROSPCustom1-46.EXE?rlkey=o9jhlzguu81jjr77uwlxcr5l1&st=5m7orsf2&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c1.46"},
}

// Gerencia o processo de instalação
type Module struct {
	TempDirBase    string
	TempDirUpdates string
	Queue          []string
}

// Cria uma nova instância do módulo Gemco Financeiro
func New(queue []string) *Module {
	return &Module{
		TempDirBase:    filepath.Join("C:\\", "TI_Setup_Temp", "GemcoFin_Base"),
		TempDirUpdates: filepath.Join("C:\\", "TI_Setup_Temp", "GemcoFin_SPs"),
		Queue:          queue,
	}
}

// Executa a fase de rede de forma assíncrona, baixando os arquivos paralelos (Fila de Download)
func (m *Module) Download(includeBase bool, wg *sync.WaitGroup) {
	defer wg.Done()

	logger.LogStep("Iniciando fase de rede do Gemco Financeiro...")
	var fila []downloader.DownloadItem

	if includeBase {
		fila = append(fila, downloader.DownloadItem{
			URL:         BaseURL,
			Destination: filepath.Join(m.TempDirBase, BaseZipName),
			Label:       "Base Financeiro",
			ExpectedMB:  50.0,
			MagicType:   "ZIP",
		})
	}

	for _, item := range m.Queue {
		if cat, exists := Catalog[item]; exists {
			fila = append(fila, downloader.DownloadItem{
				URL:         cat.URL,
				Destination: filepath.Join(m.TempDirUpdates, item),
				Label:       item,
				ExpectedMB:  0.5,
				MagicType:   "EXE",
			})
		}
	}

	err := downloader.DownloadsParalelos(fila)
	if err != nil {
		logger.WriteLog(fmt.Sprintf("Falha na Fase de Rede do Gemco Financeiro: %v", err), "ERROR", logger.Red)
	}
}

// Executa a instalação de maneira estritamente sequencial e condicional
func (m *Module) Install(includeBase bool) error {
	logger.LogStep("Iniciando Fase de Instalação Sequencial (Gemco Financeiro)")

	m.KillInstances()

	if includeBase {
		if m.IsInstalled() {
			logger.LogStep("Instalação anterior detectada. Acionando Clean Nuke Direcionado.")
			m.CleanNuke()
		}

		err := m.installBase()
		if err != nil {
			report.AddDeployReport("Gemco Financeiro", "Instalação Base (v11)", "Falha", err.Error())
			return err
		}
		report.AddDeployReport("Gemco Financeiro", "Instalação Base (v11)", "Sucesso", "")
	} else {
		if !m.IsInstalled() {
			logger.WriteLog("Base do Financeiro não detectada. Instalação isolada de SPs abortada.", "ERROR", logger.Red)
			return fmt.Errorf("base não instalada, abortando atualizações")
		}
	}

	return m.installUpdates()
}

// Verifica a existência da base do Financeiro lendo o registro ou diretório
func (m *Module) IsInstalled() bool {
	if _, err := os.Stat(BaseDir); !os.IsNotExist(err) {
		return true
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, RegPath, registry.READ)
	if err == nil {
		k.Close()
		return true
	}
	return false
}

// Derruba processos conflitantes por nome e por caminho físico (Com Polling e bloqueio de Thread para intervenção manual)
func (m *Module) KillInstances() {
	logger.LogStep("Verificando e encerrando instâncias ativas do Gemco Financeiro...")
	psScript := `
		$pendentes = [System.Collections.Generic.List[string]]::new()
		Get-Process -ErrorAction SilentlyContinue | Where-Object {
			$match = $false
			if ($null -ne $_.Path -and $_.Path.StartsWith('C:\Gemco\GemcoFinanceiro', [System.StringComparison]::OrdinalIgnoreCase)) { $match = $true }
			if ($_.ProcessName -match '(?i)gemco') { $match = $true }
			$match
		} | ForEach-Object {
			try { Stop-Process -Id $_.Id -Force -ErrorAction Stop } catch { $pendentes.Add($_.ProcessName) }
		}
		$pendentes -join ','
	`

	// Loop Interativo: Trava a execução caso os processos não possam ser mortos, esperando intervenção manual
	for {
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, _ := cmd.Output()
		pendentes := strings.TrimSpace(string(out))

		if pendentes == "" {
			break
		}

		logger.WriteLog(fmt.Sprintf("Processos retidos: %s. Encerre-os manualmente no Gerenciador de Tarefas...", pendentes), "WARNING", logger.Yellow)
		time.Sleep(20 * time.Second)
	}
}

// Erradicação completa do módulo, removendo diretórios e chaves de registro
func (m *Module) CleanNuke() {
	logger.LogStep("Executando Clean Nuke Direcionado (Gemco Financeiro)")

	uninstallPaths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}

	for _, upath := range uninstallPaths {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, upath, registry.READ)
		if err == nil {
			subkeys, _ := k.ReadSubKeyNames(-1)
			for _, sub := range subkeys {
				sk, _ := registry.OpenKey(registry.LOCAL_MACHINE, upath+`\`+sub, registry.QUERY_VALUE)
				displayName, _, _ := sk.GetStringValue("DisplayName")
				sk.Close()

				// Validação estrita dupla para apagar SOMENTE o pacote financeiro do painel de controle
				if strings.Contains(displayName, "Financeiro") && (strings.Contains(displayName, "Gemco") || strings.Contains(displayName, "TOTVS Homecenter")) {
					executor.RunSilent("msiexec.exe", "/x", sub, "/qn", "/norestart")
					installShieldPath := filepath.Join(os.Getenv("ProgramFiles(x86)"), "InstallShield Installation Information", sub)
					os.RemoveAll(installShieldPath)
					registry.DeleteKey(registry.LOCAL_MACHINE, upath+`\`+sub)
				}
			}
			k.Close()
		}
	}

	os.RemoveAll(BaseDir)
	registry.DeleteKey(registry.LOCAL_MACHINE, RegPath)
}

// Instala a base do Financeiro a partir do ZIP extraído, validando a execução e o registro pós-instalação
func (m *Module) installBase() error {
	logger.LogStep("Fase 1/2: Extraindo e Instalando Base (Pacote_v11)...")

	extractDir := filepath.Join(m.TempDirBase, "Extraido")
	zipPath := filepath.Join(m.TempDirBase, BaseZipName)

	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("falha ao extrair base: %v", err)
	}

	var setupExe string

	// Busca recursiva otimizada pelo executável mestre dentro das pastas do ZIP
	errWalk := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), "setup.exe") {
			setupExe = path
			return fmt.Errorf("Arquivo Encontrado")
		}
		return nil
	})

	if errWalk != nil && errWalk.Error() != "Arquivo Encontrado" {
		return errWalk
	}
	if setupExe == "" {
		return fmt.Errorf("setup.exe não encontrado na pasta extraída")
	}

	unblockFile(setupExe)

	exitCode, err := executor.RunSilent(setupExe, "/s", "/w", "/v/qb")
	if err != nil || (exitCode != 0 && exitCode != 3010) {
		return fmt.Errorf("falha na execução do setup (ExitCode %d): %v", exitCode, err)
	}

	if !m.IsInstalled() {
		return fmt.Errorf("validação pós-instalação no diretório/registro falhou")
	}
	return nil
}

// Instala as atualizações de SP/Custom
func (m *Module) installUpdates() error {
	totalFila := len(m.Queue)
	if totalFila == 0 {
		return nil
	}

	logger.LogStep(fmt.Sprintf("Fase 2/2: Processando Fila de Atualizações (%d itens)...", totalFila))
	sucessos := 0
	var falhas []string

	for i, item := range m.Queue {
		logger.LogStep(fmt.Sprintf("  [%d/%d] Instalando: %s", i+1, totalFila, item))

		cat, exists := Catalog[item]
		if !exists {
			continue
		}

		currVal := m.getInstalledVersion(cat.RegKey)
		if currVal == cat.RegValue {
			logger.LogStep(fmt.Sprintf("%s já está instalado. Pulando.", item))
			sucessos++
			continue
		}

		if currVal != "" {
			currWeight := getUpdateWeight(currVal)
			targetWeight := getUpdateWeight(cat.RegValue)

			if compareWeights(currWeight, targetWeight) > 0 {
				logger.LogStep("DOWNGRADE detectado: Removendo chave do registro para recuo de versão...")
				k, err := registry.OpenKey(registry.LOCAL_MACHINE, RegPath, registry.SET_VALUE)
				if err == nil {
					k.DeleteValue(cat.RegKey)
					k.Close()
				}
			}
		}

		exePath := filepath.Join(m.TempDirUpdates, item)
		unblockFile(exePath)

		var args []string
		if cat.Type == "SP" {
			args = []string{"/silent"}
		}

		exitCode, err := executor.RunSilent(exePath, args...)
		if err != nil || (exitCode != 0 && exitCode != 3010) {
			report.AddDeployReport("Gemco Financeiro", item, "Falha", fmt.Sprintf("ExitCode %d", exitCode))
			logger.WriteLog(fmt.Sprintf("Falha ao instalar %s: %v", item, err), "ERROR", logger.Red)
			falhas = append(falhas, item)
			continue
		}

		newVal := m.getInstalledVersion(cat.RegKey)
		if newVal != cat.RegValue {
			report.AddDeployReport("Gemco Financeiro", item, "Falha", "Registro não validado")
			logger.WriteLog(fmt.Sprintf("Registro de validação divergiu para %s", item), "ERROR", logger.Red)
			falhas = append(falhas, item)
			continue
		}

		report.AddDeployReport("Gemco Financeiro", item, "Sucesso", "Atualização aplicada com sucesso")
		sucessos++
	}

	// Limpeza profunda e final do cache das atualizações
	os.RemoveAll(m.TempDirUpdates)

	if len(falhas) > 0 {
		logger.WriteLog(fmt.Sprintf("Fila processada. Sucessos: %d | Falhas: %d (%s)", sucessos, len(falhas), strings.Join(falhas, ", ")), "WARNING", logger.Yellow)
	} else {
		logger.WriteLog(fmt.Sprintf("Fila 100%% processada. (%d instâncias injetadas).", sucessos), "SUCCESS", logger.Green)
	}

	return nil
}

// -----------------------------------------
// Funções Auxiliares locais
// -----------------------------------------

func (m *Module) getInstalledVersion(regKey string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, RegPath, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	val, _, err := k.GetStringValue(regKey)
	if err != nil {
		return ""
	}
	return val
}

func getUpdateWeight(versionStr string) []int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(versionStr, -1)
	var weights []int
	for _, m := range matches {
		w, _ := strconv.Atoi(m)
		weights = append(weights, w)
	}
	return weights
}

func compareWeights(a, b []int) int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	for i := 0; i < maxLen; i++ {
		valA, valB := 0, 0
		if i < len(a) {
			valA = a[i]
		}
		if i < len(b) {
			valB = b[i]
		}
		if valA < valB {
			return -1
		}
		if valA > valB {
			return 1
		}
	}
	return 0
}

// Remove identificadores de zona NTFS nativos do Windows
func unblockFile(path string) {
	os.Remove(path + ":Zone.Identifier")
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	destClean := filepath.Clean(dest)

	for _, f := range r.File {
		fpath := filepath.Join(destClean, f.Name)

		if !strings.HasPrefix(fpath, destClean+string(os.PathSeparator)) && fpath != destClean {
			return fmt.Errorf("%s: caminho de extração ZipSlip ilegal", fpath)
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
