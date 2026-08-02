package downloader

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"grc-deploy/core/logger"
	"grc-deploy/core/report"
)

// Constantes dos Magic Bytes para validação de arquivos
var (
	MagicZIP = []byte{0x50, 0x4B}
	MagicMSI = []byte{0xD0, 0xCF}
	MagicEXE = []byte{0x4D, 0x5A}
)

// O DownloadItem representa um arquivo na fila de download
type DownloadItem struct {
	URL         string
	Destination string
	Label       string
	ExpectedMB  float64
	MagicType   string
}

// Esse Result armazena o status final da operação de download
type Result struct {
	Label string
	Error error
}

// A função lê os 2 primeiros bytes do arquivo para validar o tipo de arquivo
func validarMagicBytes(filePath, magicType string) error {
	if magicType == "Nenhum" || magicType == "" {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Falha ao abrir o arquivo para validação: %v", err)
	}
	defer file.Close()

	header := make([]byte, 2)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("Falha ao ler os bytes do arquivo: %v", err)
	}

	var expected []byte
	switch magicType {
	case "ZIP":
		expected = MagicZIP
	case "MSI":
		expected = MagicMSI
	case "EXE":
		expected = MagicEXE
	default:
		return nil
	}

	if !bytes.Equal(header, expected) {
		return fmt.Errorf("magic bytes incorretos para o tipo %s. O arquivo pode estar corrompido ou é uma página HTML de erro", magicType)
	}

	return nil
}

// A função downloadworker é responsável por realizar o download de um item específico da fila de downloads
func downloadWorker(item DownloadItem, wg *sync.WaitGroup, results chan<- Result) {
	defer wg.Done()

	logger.LogStep(fmt.Sprintf("Baixando %s...", item.Label))
	logger.WriteLog(fmt.Sprintf("Iniciando download concorrente: %s -> %s", item.Label, item.URL), "INFO", logger.Cyan)

	// Cria os diretórios de destino caso não existam
	if err := os.MkdirAll(filepath.Dir(item.Destination), 0755); err != nil {
		results <- Result{Label: item.Label, Error: err}
		return
	}

	// Verificação de cache: Se o arquivo já existe, valida tamanho e magic bytes antes de decidir baixar novamente
	if info, err := os.Stat(item.Destination); err == nil {
		sizeMB := float64(info.Size()) / (1024 * 1024)

		// Verifica se o arquivo em cache tem o tamanho mínimo esperado
		if sizeMB >= item.ExpectedMB {
			// Valida a integridade (Magic Bytes) do arquivo em cache
			if err := validarMagicBytes(item.Destination, item.MagicType); err == nil {
				logger.LogSuccess(fmt.Sprintf("%s encontrado em cache local (%.2f MB). Download pulado.", item.Label, sizeMB))
				logger.WriteLog(fmt.Sprintf("Download ignorado (Cache OK): %s -> %s", item.Label, item.Destination), "INFO", logger.Cyan)
				report.AddDeployReport("Motor de Download", item.Label, report.StatusSucesso, "Download pulado (Cache validado)")

				results <- Result{Label: item.Label, Error: nil}
				return
			} else {
				logger.LogWarning(fmt.Sprintf("Cache de %s corrompido (Magic Bytes). Baixando novamente...", item.Label))
				os.Remove(item.Destination) // Remove o arquivo corrompido da pasta
			}
		} else {
			logger.LogWarning(fmt.Sprintf("Cache de %s incompleto (%.2f MB < %.2f MB). Baixando novamente...", item.Label, sizeMB, item.ExpectedMB))
			os.Remove(item.Destination) // Remove o arquivo incompleto da pasta
		}
	}

	// Realiza o download do arquivo
	resp, err := http.Get(item.URL)
	if err != nil {
		results <- Result{Label: item.Label, Error: err}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		results <- Result{Label: item.Label, Error: fmt.Errorf("falha no download, status code: %d", resp.StatusCode)}
		return
	}

	// Cria o arquivo de destino
	out, err := os.Create(item.Destination)
	if err != nil {
		results <- Result{Label: item.Label, Error: err}
		return
	}
	defer out.Close()

	// Copia o conteúdo do download para o arquivo de destino
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		results <- Result{Label: item.Label, Error: err}
		return
	}

	// Validação de tamanho do arquivo
	sizeMB := float64(written) / (1024 * 1024)
	if sizeMB < item.ExpectedMB {
		os.Remove(item.Destination)
		results <- Result{Label: item.Label, Error: fmt.Errorf("arquivo baixado é menor que o esperado: %.2f MB < %.2f MB", sizeMB, item.ExpectedMB)}
		return
	}

	//Sincronização do disco para garantir que todos os dados foram gravados
	out.Sync()

	// Validação de Magic Bytes
	if err := validarMagicBytes(item.Destination, item.MagicType); err != nil {
		os.Remove(item.Destination)
		results <- Result{Label: item.Label, Error: err}
		return
	}

	logger.LogSuccess(fmt.Sprintf("%s baixado e validado com sucesso (%.2f MB).", item.Label, sizeMB))
	logger.WriteLog(fmt.Sprintf("Download OK: %s (%.2f MB) -> %s", item.Label, sizeMB, item.Destination), "SUCCESS", logger.Green)
	report.AddDeployReport("Motor de Download", item.Label, report.StatusSucesso, "Download e validação (Magic Bytes) concluídos")

	results <- Result{Label: item.Label, Error: nil}
}

func DownloadsParalelos(items []DownloadItem) error {
	var wg sync.WaitGroup                    //inicializa o WaitGroup para sincronizar as goroutines
	results := make(chan Result, len(items)) // Cria o canal de resultados com buffer do tamanho da lista de itens

	for _, item := range items {
		wg.Add(1)
		go downloadWorker(item, &wg, results) // Inicia uma goroutine para cada item de download ("&" passa o endereço o mesmo WaitGroup para todas as goroutines)
	}

	wg.Wait()      // Aguarda todas as goroutines terminarem
	close(results) // Fecha o canal de resultados após todas as goroutines terminarem

	hasError := false
	for res := range results {
		if res.Error != nil {
			logger.WriteLog(fmt.Sprintf("Falha no download de %s: %v", res.Label, res.Error), "ERROR", logger.Red)
			report.AddDeployReport("Motor de Download", res.Label, report.StatusFalha, res.Error.Error())
			hasError = true
		}
	}
	if hasError {
		return fmt.Errorf("Falha em um ou mais downloads. Verifique os logs acima para detalhes.")
	} else {
		fmt.Println("Todos os downloads concluídos com sucesso.")
	}
	return nil
}
