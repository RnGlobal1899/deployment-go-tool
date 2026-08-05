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

import { SoftwareCard } from './components/SoftwareCard';
import { GetSoftwareModules } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

// Motor de Injeção no DOM via API-First (Consumindo do Go)
async function initDashboard() {
    const viewContainer = document.getElementById('view-container');
    if (!viewContainer) {
        console.error("[UI Error] Container 'view-container' não encontrado.");
        return;
    }

    // Limpa o placeholder tracejado do HTML estático
    viewContainer.innerHTML = '';

    try {
        // Requisita os dados reais processados pelo backend Go (Strategy Pattern)
        const modules = await GetSoftwareModules();
        
        // Renderiza cada card no grid
        modules.forEach(mod => {
            const card = new SoftwareCard(mod);
            viewContainer.appendChild(card.render());
        });
    } catch (error) {
        console.error("[IPC Error] Falha ao buscar módulos do backend:", error);
        viewContainer.innerHTML = `<div class="text-red-500 font-mono text-sm p-4 border border-red-900 bg-red-950/30 rounded-lg">Erro Crítico de IPC: ${error}</div>`;
    }
}

// 3. Orquestração de Boot do Frontend 
document.addEventListener('DOMContentLoaded', () => {
    // Inicializa a UI dinâmica
    initDashboard();

    // Escuta o listener de eventos do backend para logs em tempo real
    EventsOn('terminal-log', (logLine: string, colorCode: string) => {
        const cleanLine = logLine.trimEnd();

        // Mapeia o colorCode do backend para classes Tailwind
        let colorClass = 'text-slate-400';
        if (colorCode.includes('31m')) colorClass = 'text-red-500';
        if (colorCode.includes('32m')) colorClass = 'text-green-400';
        if (colorCode.includes('33m')) colorClass = 'text-amber-400';
        if (colorCode.includes('36m')) colorClass = 'text-cyan-500';

        const logEntry = document.createElement('div');
        logEntry.className = `font-mono text-sm ${colorClass}`;
        logEntry.textContent = cleanLine;

        terminalOutput.appendChild(logEntry);
        terminalOutput.scrollTop = terminalOutput.scrollHeight;
    });

    logToTerminal('Híbrido Go + Wails carregado. Iniciando UI Dark Tech...', 'INFO');

});

// Inicialização
//document.addEventListener('DOMContentLoaded', () => {
    //logToTerminal('DOM Carregado. Construindo UI Dark Tech...', 'INFO');
//});

