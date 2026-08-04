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

import './style.css'; 
import { SoftwareCard, SoftwareModule } from './components/SoftwareCard';

// 1. Catálogo Simulado (Mock) para validação visual das Accent Colors e SVGs
const mockModules: SoftwareModule[] = [
    { 
        id: 'kaspersky', 
        name: 'Kaspersky Endpoint', 
        description: 'Orquestração de rede e segurança de endpoint. Instalação silenciosa.', 
        iconSvg: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z" /></svg>' 
    },
    { 
        id: 'wps', 
        name: 'WPS Office', 
        description: 'Suíte de produtividade padronizada GRC.', 
        iconSvg: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" /></svg>' 
    },
    { 
        id: 'gemco-fin', 
        name: 'Gemco Financeiro', 
        description: 'ERP com regras isoladas (Clean Nuke direcionado e Polling).', 
        iconSvg: '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 6v12m-3-2.818.879.659c1.171.879 3.07.879 4.242 0 1.172-.879 1.172-2.303 0-3.182C13.536 12.219 12.768 12 12 12c-.725 0-1.45-.22-2.003-.659-1.106-.879-1.106-2.303 0-3.182s2.9-.879 4.006 0l.415.33M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" /></svg>' 
    }
];

// 2. Motor de Injeção no DOM
function initDashboard() {
    const viewContainer = document.getElementById('view-container');
    if (!viewContainer) {
        console.error("[UI Error] Container 'view-container' não encontrado.");
        return;
    }

    // Limpa o placeholder tracejado do HTML estático
    viewContainer.innerHTML = '';

    // Renderiza cada card do catálogo
    mockModules.forEach(mod => {
        const card = new SoftwareCard(mod);
        viewContainer.appendChild(card.render());
    });
}

// 3. Orquestração de Boot do Frontend
document.addEventListener('DOMContentLoaded', () => {
    // Inicializa a UI dinâmica
    initDashboard();
});

// Inicialização
//document.addEventListener('DOMContentLoaded', () => {
    //logToTerminal('DOM Carregado. Construindo UI Dark Tech...', 'INFO');
//});

