package gemco

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

	"grc-deploy/core/downloader"
	"grc-deploy/core/executor"
	"grc-deploy/core/logger"
	"grc-deploy/core/report"

	"golang.org/x/sys/windows/registry"
)

// Constantes fundamentais para o modulo
const (
	BaseZipName  = "Gemco_Base.zip"
	BaseURL      = "https://www.dropbox.com/scl/fo/lo1er05dmtg2m6jmkcmh0/AAv7_yJz2xqKBnfSTb10SuE?rlkey=9uonw6qv5e0zlzm3occis6jfv&st=g19zb4gn&dl=1"
	OdbcmZipName = "ODBCM_Setup.zip"
	OdbcmURL     = "https://www.dropbox.com/scl/fo/wyy74lzab2qh8vcl54q9f/AI1tURttIbOfG9MPWVY82YM?rlkey=z3ok3m4b8el4zafxlufo0w82m&st=z76lip6y&dl=1"
	BaseDir      = "C:\\Gemco"
	OdbcmDestDir = "C:\\Gemco\\ODBCM"
	RegPath      = `SOFTWARE\WOW6432Node\Gemco\Gemco2000`
	RegPathRoot  = `SOFTWARE\WOW6432Node\Gemco`
)

// Struct para definir a estrutura das atualizações do Gemco
type UpdateCatalog struct {
	URL      string
	Type     string
	RegKey   string
	RegValue string
}

// Catálogo Global
var Catalog = map[string]UpdateCatalog{
	"Gemco2002SP39-00-37.exe":     {URL: "https://www.dropbox.com/scl/fi/51ztqbx038m77n4impwwl/Gemco2002SP39-00-37.exe?rlkey=96pquc63it54256qpsrc9old5&st=fvy51cgf&dl=1", Type: "SP", RegKey: "SP", RegValue: "SP39.00.37"},
	"Gemco2002SP44-00-121.exe":    {URL: "https://www.dropbox.com/scl/fi/1xwc5od5gzk3ruh67rudh/Gemco2002SP44-00-121.exe?rlkey=4b54ybb7eau20f5yhx45meq2z&st=hc6ccnmc&dl=1", Type: "SP", RegKey: "SP", RegValue: "SP44.00.121"},
	"Gemco2002SP44-00-121-68.exe": {URL: "https://www.dropbox.com/scl/fi/7ydebpvwgbmly5a0lacel/Gemco2002SP44-00-121-68.exe?rlkey=kt0u9pohdqrms0jmpt32n7eox&st=kqlb80hj&dl=1", Type: "SP", RegKey: "SP", RegValue: "SP44.00.121.68"},
	"Gemco2002SP44-00-121-75.exe": {URL: "https://www.dropbox.com/scl/fi/ebxtxyumilfcg9v1dbyn8/Gemco2002SP44-00-121-75.exe?rlkey=i7lthgy7sb4l5udn2vcvc10ce&st=9tzj2go4&dl=1", Type: "SP", RegKey: "SP", RegValue: "SP44.00.121.75"},
	"Custom11-86.EXE":             {URL: "https://www.dropbox.com/scl/fi/7h7hz5cnglp6zjykn4u15/Custom11-86.EXE?rlkey=1u4c7soaay0m3tvt19gigs2nf&st=0fslioqs&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c11.86"},
	"Gemco2002SPCustom11-09.EXE":  {URL: "https://www.dropbox.com/scl/fi/r2hm0zhinpuro5u1uupfv/Gemco2002SPCustom11-09.EXE?rlkey=g11rcqr6n6wnkjhbyvm3vjk33&st=skoonh5u&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c11.09"},
	"Gemco2002SPCustom11-16.EXE":  {URL: "https://www.dropbox.com/scl/fi/1vaglhq1pdc2o1qay8w1t/Gemco2002SPCustom11-16.EXE?rlkey=bpjn3po1sum5zt5pzi2kbqqs8&st=o87icob6&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c11.16"},
	"Gemco2002SPCustom11-26.EXE":  {URL: "https://www.dropbox.com/scl/fi/okg0xxgmwh6jbl8c94slu/Gemco2002SPCustom11-26.EXE?rlkey=gvz0odk1lqyn4riajfyzoh8gy&st=vi6r670f&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c11.26"},
	"Gemco2002SPCustom11-34.EXE":  {URL: "https://www.dropbox.com/scl/fi/0xfe3jpniptxq72qweudx/Gemco2002SPCustom11-34.EXE?rlkey=9jnumwjwm4wkki6xp66n12ba1&st=e8h4dsx7&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c11.34"},
	"Gemco2002SPCustom11-77.EXE":  {URL: "https://www.dropbox.com/scl/fi/b10slhqds8yz9a2xmbjrl/Gemco2002SPCustom11-77.EXE?rlkey=wqhw4qekcx8tndk5d5j23ysrq&st=d1vnzku1&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c11.77"},
	"Gemco2002SPCustom11-80.EXE":  {URL: "https://www.dropbox.com/scl/fi/l86dp8q4qz2eydtwpnq1p/Gemco2002SPCustom11-80.EXE?rlkey=gjaf8nnrrocdxxb2noukg06xd&st=2pcvborw&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c11.80"},
	"Gemco2002SPCustom12-12.EXE":  {URL: "https://www.dropbox.com/scl/fi/zzvsh3xpu5hblfmdligex/Gemco2002SPCustom12-12.EXE?rlkey=74jz4cyob7oeb86h5pmfivl5r&st=u7zxq6rb&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.12"},
	"Gemco2002SPCustom12-18.EXE":  {URL: "https://www.dropbox.com/scl/fi/jg0fmlwpnnsj7kje15vy1/Gemco2002SPCustom12-18.EXE?rlkey=75pgv74laj31pfv0h1hjnfjnu&st=k1q0gbsi&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.18"},
	"Gemco2002SPCustom12-27.EXE":  {URL: "https://www.dropbox.com/scl/fi/23yfof097c5ups1egj5l5/Gemco2002SPCustom12-27.EXE?rlkey=clpr8jxdj8prpuw4kdshxlm3u&st=nv57012e&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.27"},
	"Gemco2002SPCustom12-95.EXE":  {URL: "https://www.dropbox.com/scl/fi/i4s0t8ng44y1ckkez6i4q/Gemco2002SPCustom12-95.EXE?rlkey=s1t05xthdp1piyshbm8xcvl7s&st=is2rm8s2&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.95"},
	"Gemco2002SPCustom12-111.EXE": {URL: "https://www.dropbox.com/scl/fi/fxkg1nd4oxw7avnoccgdw/Gemco2002SPCustom12-111.EXE?rlkey=4uf9c7rougt8x03uz09f774fy&st=0zkdzfqw&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.111"},
	"Gemco2002SPCustom12-125.EXE": {URL: "https://www.dropbox.com/scl/fi/9oasebodbgszen0fft5ge/Gemco2002SPCustom12-125.EXE?rlkey=6xu1h47uqk5b75priy9lbylo3&st=s8we9gr3&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.125"},
	"Gemco2002SPCustom12-156.exe": {URL: "https://www.dropbox.com/scl/fi/938mihixpjrhznwjuquac/Gemco2002SPCustom12-156.exe?rlkey=3a36w6jp6shhhw26ydi6a116u&st=iok2zhzb&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.156"},
	"Gemco2002SPCustom12-164.EXE": {URL: "https://www.dropbox.com/scl/fi/eg0r530ofnvzm3x5lumcz/Gemco2002SPCustom12-164.EXE?rlkey=1a70yflb64rmdjnwyqdldwhk3&st=0ks4zo6m&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.164"},
	"Gemco2002SPCustom12-213.EXE": {URL: "https://www.dropbox.com/scl/fi/ptinb7dla1cghlpjgxb50/Gemco2002SPCustom12-213.EXE?rlkey=4ccr0c7zy7p0ut1laeyylkdmu&st=i3l10lzr&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.213"},
	"Gemco2002SPCustom12-219.EXE": {URL: "https://www.dropbox.com/scl/fi/0fqjzp2m3pib8xp7anhhk/Gemco2002SPCustom12-219.EXE?rlkey=cvwmpmxmdrmzzevaryvmmvkd5&st=kxy4lkc8&dl=1", Type: "Custom", RegKey: "SPCustom", RegValue: "c12.219"},
}

// Struct para gerenciar o processo de instalação do Gemco
type Module struct {
	TempDir string
	Queue   []string
}

// Auxiliar: Lê a versão atual instalada no registro
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

// Auxiliar: Extrai os pesos númericos das versões para calculo seguro
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

// Auxiliar: Compara o peso de duas versões
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

// Auxiliar: Injeta o SP 121 dinamicamente na fila caso o SP alvo requeira
func (m *Module) applyQueueBusinessRules() {
	// Verifica se existe a SP121 no catalogo
	sp121Key := "Gemco2002SP44-00-121.exe"
	sp121Cat, exists := Catalog[sp121Key]
	if !exists {
		return
	}
	// Pega o valor bruto da SP
	sp121Weight := getUpdateWeight(sp121Cat.RegValue)

	// Pega a versão atual instalada e seu peso
	currSP := m.getInstalledVersion("SP")
	currWeight := getUpdateWeight(currSP)

	// Analisa a fila atual para encontrar o primeiro pacote que exija a SP 121
	inject121Index := -1
	isDowngradeThatNeeds121 := false

	for i, item := range m.Queue {
		cat, ok := Catalog[item]
		if ok && cat.Type == "SP" {
			targetWeight := getUpdateWeight(cat.RegValue)
			if compareWeights(targetWeight, sp121Weight) > 0 {
				inject121Index = i

				// Se o ambiente atual já for maior que a SP alvo, caracteriza um downgrade
				if currSP != "" && compareWeights(currWeight, targetWeight) > 0 {
					isDowngradeThatNeeds121 = true
				}
				break
			}
		}
	}

	// Se encontrou um pacote que exige a SP 121, verifica se ela já está na fila.
	// se não estiver, injeta a SP 121 antes do pacote que a requer.
	if inject121Index >= 0 {
		contains121 := false
		for _, item := range m.Queue {
			if item == sp121Key {
				contains121 = true
				break
			}
		}

		if !contains121 {
			// A injeção ocorre se: for uma base nova (currSP == ""), for upgrade vindo de SP < 121,
			// ou for downgrade que reseta a versão
			if currSP == "" || compareWeights(currWeight, sp121Weight) < 0 || isDowngradeThatNeeds121 {
				newQueue := make([]string, 0, len(m.Queue)+1)
				newQueue = append(newQueue, m.Queue[:inject121Index]...)
				newQueue = append(newQueue, sp121Key)
				newQueue = append(newQueue, m.Queue[inject121Index:]...)
				m.Queue = newQueue
			}
		}
	}
}

// Cria uma nova instância do módulo Gemco
func New(queue []string) *Module {
	tempPath := filepath.Join("C:\\", "TI_Setup_Temp", "Gemco_Base")
	m := &Module{
		TempDir: tempPath,
		Queue:   queue,
	}
	m.applyQueueBusinessRules()
	return m
}

// Executa a fase de rede de forma assíncrona, baixando os arquivos necessários
func (m *Module) Download(includeBase bool, wg *sync.WaitGroup) {
	defer wg.Done()

	logger.LogStep("Iniciando fase de download do Gemco...")

	var fila []downloader.DownloadItem

	if includeBase {
		// Download da base do Gemco
		fila = append(fila, downloader.DownloadItem{
			URL:         BaseURL,
			Destination: filepath.Join(m.TempDir, BaseZipName),
			Label:       "Gemco Base",
			ExpectedMB:  100.0,
			MagicType:   "ZIP",
		})

		// Download do ODBCM
		fila = append(fila, downloader.DownloadItem{
			URL:         OdbcmURL,
			Destination: filepath.Join(m.TempDir, OdbcmZipName),
			Label:       "Utilitário ODBCM",
			ExpectedMB:  0.3,
			MagicType:   "ZIP",
		})
	}
	// Download da Fila de SPs/Customs
	for _, item := range m.Queue {
		if cat, exists := Catalog[item]; exists {
			fila = append(fila, downloader.DownloadItem{
				URL:         cat.URL,
				Destination: filepath.Join(m.TempDir, item),
				Label:       item,
				ExpectedMB:  0.5,
				MagicType:   "EXE",
			})
		}
	}

	err := downloader.DownloadsParalelos(fila)
	if err != nil {
		logger.WriteLog(fmt.Sprintf("Falha na Fase de Rede do Gemco: %v", err), "ERROR", logger.Red)
	}
}

// Executa a instalação de maneira sequencial
func (m *Module) Install(includeBase bool) error {
	logger.LogStep("Iniciando Fase de Instalação Sequencial (Gemco 2002)")

	m.KillInstances()

	if includeBase {
		if m.IsInstalled() {
			logger.LogStep("Base existente detectada. Acionando Clean Nuke.")
			m.CleanNuke()
		}

		err := m.installBase()
		if err != nil {
			report.AddDeployReport("Gemco 2002", "Instalação Base", "Falha", err.Error())
			return err
		}
		report.AddDeployReport("Gemco 2002", "Instalação Base", "Sucesso", "")

		err = m.installODBCM()
		if err != nil {
			report.AddDeployReport("Gemco 2002", "Configuração ODBCM", "Falha", err.Error())
			return err
		}
	} else {
		if !m.IsInstalled() {
			logger.WriteLog("Gemco Base não detectada. A instalação isolada de SPs/Customs pode falhar.", "WARNING", logger.Yellow)
		}
	}

	return m.installUpdates()
}

// Verifica a existência da base, verificando o registro
func (m *Module) IsInstalled() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, RegPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetStringValue("Diretorio_Padrao")
	return err == nil && val != ""
}

// Derruba processos conflitantes
func (m *Module) KillInstances() {
	procs := []string{"gemco.exe", "gconfig.exe", "odbcm.exe"}
	for _, p := range procs {
		cmd := exec.Command("taskkill", "/F", "/IM", p, "/T")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
	}
}

// Limpa profundamente as instalações do Gemco detectadas
func (m *Module) CleanNuke() error {
	logger.LogStep("Executando Clean Nuke Extremo (Gemco 2002)")

	// 1. Chamada para matar os processos
	m.KillInstances()

	// 2. Localiza e desinstala via Registro (WOW6432Node e Padrão)
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

				if strings.Contains(displayName, "Gemco") || strings.Contains(displayName, "TOTVS Homecenter") {
					if !strings.Contains(displayName, "Financeiro") {
						// Aciona o desinstalador silencioso do MSI
						executor.RunSilent("msiexec.exe", "/x", sub, "/qn", "/norestart")

						// Erradica o Cache Oculto do InstallShield
						installShieldPath := filepath.Join(os.Getenv("ProgramFiles(x86)"), "InstallShield Installation Information", sub)
						os.RemoveAll(installShieldPath)

						// Erradica a chave do Gemco no regedit
						registry.DeleteKey(registry.LOCAL_MACHINE, upath+`\`+sub)
					}
				}
			}
			k.Close()
		}
	}

	// 3. Remove diretório físico (Preservando GemcoFinanceiro)
	entries, err := os.ReadDir(BaseDir)
	if err == nil {
		for _, entry := range entries {
			if !strings.EqualFold(entry.Name(), "GemcoFinanceiro") {
				os.RemoveAll(filepath.Join(BaseDir, entry.Name()))
			}
		}
	}

	// 4. Limpa chaves de registro raiz e arquivos INI globais
	registry.DeleteKey(registry.LOCAL_MACHINE, RegPathRoot)
	os.Remove("C:\\Windows\\gemco.ini")
	os.Remove("C:\\Windows\\gemcoci.ini")

	return nil
}

// Inicia o configurador do Gemco
func (m *Module) InitGconfig() error {
	gconfigPath := filepath.Join(BaseDir, "ActiveX", "Exe", "Gconfig.exe")
	if _, err := os.Stat(gconfigPath); os.IsNotExist(err) {
		return fmt.Errorf("gconfig.exe não encontrado no diretório: %s", gconfigPath)
	}

	logger.LogStep("Iniciando Gconfig para configuração manual via UI.")
	cmd := exec.Command(gconfigPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	return cmd.Start()
}

// Métodos Privados da Pipeline de Instalação

func (m *Module) installBase() error {
	logger.LogStep("Fase 1/3: Extraindo e Instalando Gemco Base...")

	extractDir := filepath.Join(m.TempDir, "Extraido")
	zipPath := filepath.Join(m.TempDir, BaseZipName)

	// Extração baseada em helper do GO
	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("falha ao extrair base: %v", err)
	}

	setupExe := filepath.Join(extractDir, "setup.exe")

	// Execução da instalação da base ("/w" bloqueia a thread e aguarda o MSI "filho" encerrar)
	exitCode, err := executor.RunSilent(setupExe, "/s", "/w", "/v/passive")
	if err != nil || (exitCode != 0 && exitCode != 3010) {
		return fmt.Errorf("Falha na execução do setup.exe (ExitCode %d): %v", exitCode, err)
	}

	if !m.IsInstalled() {
		return fmt.Errorf("base não validada no registro após instalação")
	}
	return nil
}

func (m *Module) installODBCM() error {
	zipPath := filepath.Join(m.TempDir, OdbcmZipName)
	os.MkdirAll(OdbcmDestDir, os.ModePerm)

	if err := extractZip(zipPath, OdbcmDestDir); err != nil {
		return fmt.Errorf("falha ao extrair o utilitário ODBCM: %v", err)
	}

	odbcmExe := filepath.Join(OdbcmDestDir, "ODBCM.exe")

	// Configura o "AppCompatFlags" para rodar ODBCM silenciosamente como Administrador
	k, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\AppCompatFlags\Layers`, registry.SET_VALUE)
	if k != 0 {
		defer k.Close()
		k.SetStringValue(odbcmExe, "~ RUNASADMIN")
	}

	logger.LogStep("Fase 2/3: Configurando e inicializando Utilitário ODBCM...")

	// Dispara e continua de maneira assincrona
	if err := executor.RunAsync(odbcmExe); err != nil {
		logger.WriteLog(fmt.Sprintf("Falha ao iniciar ODBCM.exe assincronamente: %v", err), "ERROR", logger.Red)
	}

	return nil
}

func (m *Module) installUpdates() error {
	if len(m.Queue) > 0 {
		logger.LogStep(fmt.Sprintf("Fase 3/3: Processando Fila de Atualizações Sequencial (%d itens)...", len(m.Queue)))
	}
	// O loop executa de forma sequencial na ordem inserida pelo Frontend.
	for i, item := range m.Queue {
		logger.LogStep(fmt.Sprintf("  [%d/%d] Instalando pacote: %s", i+1, len(m.Queue), item))

		// Verifica se o item existe no catalogo
		cat, exists := Catalog[item]
		if !exists {
			continue
		}

		// Verifica a versão atual instalada
		currVal := m.getInstalledVersion(cat.RegKey)
		if currVal == cat.RegValue {
			logger.LogStep(fmt.Sprintf("%s já instalado. Pulando.", item))
			continue
		}

		// Se o valor atual não for nulo, compara ambas as versões para realizar downgrade se necessário
		if currVal != "" {
			currWeight := getUpdateWeight(currVal)
			targetWeight := getUpdateWeight(cat.RegValue)

			if compareWeights(currWeight, targetWeight) > 0 {
				logger.LogStep("DOWNGRADE Detectado: Preparando ambiente para versão divergente...")
				k, err := registry.OpenKey(registry.LOCAL_MACHINE, RegPath, registry.SET_VALUE)
				if err == nil {
					k.DeleteValue(cat.RegKey)
					k.Close()
				}

				// Sendo custom, remove uma dll que fica constantemente como resquicio
				if cat.Type == "Custom" {
					resquicioDll := filepath.Join(BaseDir, "ActiveX", "Custom", "DLL", "ccgstConsultaPedido.dll")
					os.Remove(resquicioDll)
				}
			}
		}

		exePath := filepath.Join(m.TempDir, item)

		// Define o tipo de argumento baseado no tipo do item do catalogo (custom não leva argumento)
		var args []string
		if cat.Type == "SP" {
			args = []string{"/silent"}
		}

		// Executa a atualização
		exitCode, err := executor.RunSilent(exePath, args...)
		if err != nil || (exitCode != 0 && exitCode != 3010) {
			report.AddDeployReport("Gemco 2002", item, "Falha", fmt.Sprintf("ExitCode: %d", exitCode))
			logger.WriteLog(fmt.Sprintf("Falha ao instalar %s (ExitCode %d): %v", item, exitCode, err), "ERROR", logger.Red)
			continue
		}

		// Validação pós-instalação: Verifica se o valor do registro foi atualizado corretamente
		newVal := m.getInstalledVersion(cat.RegKey)
		if newVal != cat.RegValue {
			report.AddDeployReport("Gemco 2002", item, "Falha", fmt.Sprintf("Registro não atualizado: %s", newVal))
			logger.WriteLog(fmt.Sprintf("Falha na validação pós-instalação de %s: Registro não atualizado (Esperado: %s, Atual: %s)", item, cat.RegValue, newVal), "ERROR", logger.Red)
			continue
		}

		report.AddDeployReport("Gemco 2002", item, "Sucesso", "")
		logger.LogStep(fmt.Sprintf("%s instalado com sucesso.", item))
	}

	return nil
}

// Extrai arquivos .zip de forma segura, prevenindo vulnerabilidades de path traversal (ZipSlip)
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
