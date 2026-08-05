package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var logMu sync.Mutex
var wailsCtx context.Context

// Injeta o contexto do Wails para permitir a comunicação com o frontend
func SetContext(ctx context.Context) {
	wailsCtx = ctx
}

// Códigos ANSI para colorir o console
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
)

// Váriavel que guarda o caminho do arquivo de log
var logFilePath string

// Função para definitir onde o log será salvo
func InitLogger(TempFolder string) {
	// Cria o diretório de logs se não existir
	os.MkdirAll(TempFolder, os.ModePerm)
	logFilePath = filepath.Join(TempFolder, "deployment_tool.txt")
}

// Função para escrever os passos
func LogStep(msg string) {
	fmt.Printf("\n >> %s%s%s\n", Cyan, msg, Reset)
}

// Função para escrever sucesso
func LogSuccess(msg string) {
	fmt.Printf(" [OK] >> %s%s%s\n", Green, msg, Reset)
}

// Função para escrever aviso
func LogWarning(msg string) {
	fmt.Printf(" [!] >> %s%s%s\n", Yellow, msg, Reset)
}

// WriteLog grava no arquivo de texto e imprime no console simultaneamente
func WriteLog(msg, level, colorCode string) {
	// Formata a data (Go usa uma data de referência fixa "2006-01-02 15:04:05" para formatação)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)

	// Output visual no console
	fmt.Printf("%s%s%s", colorCode, logLine, Reset)

	logMu.Lock()         // Bloqueia o acesso ao arquivo de log
	defer logMu.Unlock() // Garante que o mutex será desbloqueado ao final da função

	// Grava no arquivo de log
	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("%s[ERROR] Falha ao abrir arquivo de log: %v%s\n", Red, err, Reset)
		return
	}
	defer f.Close() // Garante que o arquivo será fechado após a escrita

	f.WriteString(logLine)

	// Envia o log para o frontend via Wails
	if wailsCtx != nil {
		// Emite o evento "terminal-log" para o frontend com a mensagem de log e a cor
		runtime.EventsEmit(wailsCtx, "terminal-log", logLine, colorCode)
	}
}
