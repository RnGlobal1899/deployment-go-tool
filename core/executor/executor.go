package executor

import (
	"os/exec"
	"syscall"
)

// Executa um binário de forma sincrona, interceptando o código de saída e usando SysProcAttr para evitar a criação de uma janela no Windows.
// "..." permite passar múltiplos argumentos para o comando.
func RunSilent(executable string, args ...string) (int, error) {
	cmd := exec.Command(executable, args...)

	// Chamada de baixo nível para a Win32 API garantindo execução invisível
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	err := cmd.Run()
	if err != nil {
		// Tenta capturar o código de saída do processo
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), err
		}
		return -1, err
	}

	// Se o comando foi bem-sucedido, retorna 0
	return 0, nil
}

// Executa um binário de forma assíncrona, sem esperar pelo término do processo e sem criar uma janela no Windows.
func RunAsync(executable string, args ...string) error {
	cmd := exec.Command(executable, args...)

	// Chamada de baixo nível para a Win32 API garantindo execução invisível
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
