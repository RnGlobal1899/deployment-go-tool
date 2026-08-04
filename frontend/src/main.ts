import './style.css';

// Variáveis globais para manipular o DOM
const terminalOutput = document.getElementById('terminal-output') as HTMLDivElement;

/**
 * Utilitário provisório para escrever no console inferior da interface
 */
export function logToTerminal(message: string, level: 'INFO' | 'SUCCESS' | 'WARNING' | 'ERROR' = 'INFO') {
    const now = new Date();
    const timeString = now.toTimeString().split(' ')[0]; // Ex: 14:35:00
    
    let colorClass = 'text-slate-400';
    switch (level) {
        case 'SUCCESS': colorClass = 'text-green-400'; break;
        case 'WARNING': colorClass = 'text-amber-400'; break;
        case 'ERROR':   colorClass = 'text-red-500'; break;
        case 'INFO':    colorClass = 'text-cyan-500'; break;
    }

    const logEntry = document.createElement('div');
    logEntry.className = `flex items-start space-x-3 ${colorClass}`;
    logEntry.innerHTML = `
        <span class="text-slate-600">[${timeString}]</span>
        <span>[${level}] ${message}</span>
    `;

    terminalOutput.appendChild(logEntry);
    
    // Rola o terminal automaticamente para o fundo
    terminalOutput.scrollTop = terminalOutput.scrollHeight;
}

// Inicialização
document.addEventListener('DOMContentLoaded', () => {
    logToTerminal('DOM Carregado. Construindo UI Dark Tech...', 'INFO');
});