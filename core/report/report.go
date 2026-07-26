package report

import (
	"sync"
)

// Criação de um type customizado para guardar as opções de Status (Sucesso, Falha, etc.)
type Status string

const (
	StatusSucesso Status = "Sucesso"
	StatusFalha   Status = "Falha"
	StatusAviso   Status = "Aviso"
)

// Define a estrutura dos dados
type ReportItem struct {
	Componente string
	Tarefa     string
	Status     Status
	Detalhes   string
}

var (
	// Array global para armazenar os itens do relatório
	relatorioDeploy []ReportItem
	// Mutex para garantir que apenas uma goroutine acesse o array de relatório por vez
	mu sync.Mutex
)

// Interface pública para adicionar itens ao relatório
func AddDeployReport(Componente, Tarefa string, Status Status, Detalhes string) {
	mu.Lock()         // Bloqueia o acesso ao array de relatório
	defer mu.Unlock() // Garante que o mutex será desbloqueado ao final da função

	item := ReportItem{
		Componente: Componente,
		Tarefa:     Tarefa,
		Status:     Status,
		Detalhes:   Detalhes,
	}

	relatorioDeploy = append(relatorioDeploy, item)
}

// Retorna uma cópia do relatório de deploy
func GetReport() []ReportItem {
	mu.Lock()         // Bloqueia o acesso ao array de relatório
	defer mu.Unlock() // Garante que o mutex será desbloqueado ao final da função

	// Retorna uma cópia do relatório para evitar que o array original seja modificado externamente
	copia := make([]ReportItem, len(relatorioDeploy))
	copy(copia, relatorioDeploy)

	return copia
}
