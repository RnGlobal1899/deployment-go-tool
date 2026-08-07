import * as AppIPC from '../../wailsjs/go/main/App';

const CheckGemcoFinanceiroStatus = (AppIPC as any).CheckGemcoFinanceiroStatus || (async () => false);
const GetGemcoFinanceiroCatalog = (AppIPC as any).GetGemcoFinanceiroCatalog || (async () => []);
const ExecuteGemcoFinanceiroWizard = (AppIPC as any).ExecuteGemcoFinanceiroWizard || (async () => {});

export class GemcoFinanceiroWizard {
    private root: HTMLElement;
    private content!: HTMLElement;
    private btnNext!: HTMLButtonElement;
    
    private step: number = 1;
    private isInstalled: boolean = false;
    private availableUpdates: string[] = [];
    
    private selectedAction: string = "";
    private selectedQueue: string[] = [];

    constructor() {
        this.root = document.getElementById('modal-root') as HTMLElement;
        const template = document.getElementById('gemco-financeiro-wizard-template') as HTMLTemplateElement;
        
        if (!template) {
            console.error("[UI Error] Template 'gemco-financeiro-wizard-template' ausente.");
            return;
        }

        this.root.innerHTML = '';
        this.root.appendChild(template.content.cloneNode(true));
        
        this.content = this.root.querySelector('.wizard-content') as HTMLElement;
        this.btnNext = this.root.querySelector('.btn-next') as HTMLButtonElement;
        const btnCancel = this.root.querySelector('.btn-cancel') as HTMLButtonElement;

        btnCancel.addEventListener('click', () => this.unmount());
        this.btnNext.addEventListener('click', () => this.handleNextStep());
    }

    public async mount() {
        this.root.classList.remove('hidden');
        this.root.classList.add('flex');
        
        this.btnNext.textContent = "Verificando Ambiente...";
        this.btnNext.disabled = true;

        // Requisições IPC Simultâneas
        const [status, catalog] = await Promise.all([
            CheckGemcoFinanceiroStatus(),
            GetGemcoFinanceiroCatalog()
        ]);

        this.isInstalled = status;
        this.availableUpdates = catalog || [];

        this.btnNext.disabled = false;
        this.renderStep1();
    }

    private unmount() {
        this.root.classList.add('hidden');
        this.root.classList.remove('flex');
        this.root.innerHTML = '';
    }

    private async handleNextStep() {
        if (this.step === 1) {
            this.selectedAction = (this.content.querySelector('#gemco-action') as HTMLSelectElement).value;

            if (this.selectedAction === 'skip') {
                this.unmount();
                return;
            }
            if (this.selectedAction === 'uninstall') {
                this.step = 3;
                this.renderStep3();
                return;
            }

            // Se for install_full ou update_only, avança para seleção de fila
            this.step = 2;
            this.renderStep2();
            return;
        }

        if (this.step === 2) {
            // Varre o DOM procurando checkboxes checados
            const checkboxes = this.content.querySelectorAll('.update-cb:checked');
            this.selectedQueue = Array.from(checkboxes).map(cb => (cb as HTMLInputElement).value);
            
            this.step = 3;
            this.renderStep3();
            return;
        }

        if (this.step === 3) {
            this.btnNext.textContent = "Iniciando Orquestração...";
            this.btnNext.disabled = true;

            const includeBase = this.selectedAction === 'install_full';
            const uninstallOnly = this.selectedAction === 'uninstall';
            
            // Repassa as flags e a fila selecionada (mesmo que vazia) para o Go
            await ExecuteGemcoFinanceiroWizard(includeBase, this.selectedQueue, uninstallOnly);
            this.unmount();
        }
    }

    private renderStep1() {
        this.btnNext.textContent = "Avançar";

        const statusBadge = this.isInstalled 
            ? `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-cyan-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-cyan-500 animate-pulse"></span>
                  <span>Base do Gemco Fin Localizada</span>
               </div>`
            : `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-slate-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-slate-500"></span>
                  <span>Módulo Inexistente nesta Máquina</span>
               </div>`;

        const options = this.isInstalled
            ? `<option value="update_only">Atualização Incremental (Apenas SPs/Customs)</option>
               <option value="install_full">Clean Install (Forçar reinstalação completa)</option>
               <option value="uninstall">Clean Nuke (Erradicar Base e Registros)</option>
               <option value="skip">Cancelar Ação</option>`
            : `<option value="install_full">Instalação Limpa Completa</option>
               <option value="skip">Cancelar Ação</option>`;

        this.content.innerHTML = `
            <div class="space-y-4">
                ${statusBadge}
                <div class="space-y-2">
                    <label class="block text-slate-400 font-medium">Comando de Orquestração:</label>
                    <select id="gemco-action" class="w-full bg-slate-950 border border-slate-700 text-slate-200 rounded-lg p-2.5 outline-none focus:border-cyan-500 transition-colors">
                        ${options}
                    </select>
                </div>
                <p class="text-xs text-slate-500 font-mono border-l-2 border-slate-700 pl-2 mt-2">
                    Operações Clean Nuke realizam 'Process Kill' e varredura de chaves orfãs no regedit de forma autônoma.
                </p>
            </div>
        `;
    }

    private renderStep2() {
        this.btnNext.textContent = "Confirmar Fila";

        let checkboxesHTML = this.availableUpdates.map(update => `
            <label class="flex items-center space-x-3 p-3 bg-slate-950 border border-slate-800 rounded-lg cursor-pointer hover:border-cyan-500/50 transition-colors">
                <input type="checkbox" value="${update}" class="update-cb w-4 h-4 text-cyan-600 bg-slate-900 border-slate-700 rounded focus:ring-cyan-500 focus:ring-2">
                <span class="text-sm font-mono text-slate-300">${update}</span>
            </label>
        `).join('');

        if (this.availableUpdates.length === 0) {
            checkboxesHTML = `<div class="text-xs text-amber-500 font-mono">Nenhum pacote catalogado no backend.</div>`;
        }

        this.content.innerHTML = `
            <div class="space-y-3">
                <label class="block text-slate-400 font-medium mb-2">Montagem da Fila (Updates):</label>
                <div class="space-y-2 max-h-48 overflow-y-auto pr-2 custom-scrollbar">
                    ${checkboxesHTML}
                </div>
            </div>
        `;
    }

    private renderStep3() {
        this.btnNext.textContent = "Confirmar Deploy Win32";
        
        let actionDesc = "N/A";
        let filaDesc = this.selectedQueue.length > 0 ? this.selectedQueue.join('<br>+ ') : "Nenhuma (Apenas Base)";

        if (this.selectedAction === 'install_full') actionDesc = "Package Download -> Clean Nuke (Se houver) -> Base Install -> Queue Processing";
        if (this.selectedAction === 'update_only') actionDesc = "Package Download -> Queue Processing";
        if (this.selectedAction === 'uninstall') {
            actionDesc = "Process Kill (Gemco) -> Clean Nuke (Arquivos & Regedit)";
            filaDesc = "Ignorada";
        }

        this.content.innerHTML = `
            <div class="space-y-3 bg-slate-950 border border-slate-800 p-4 rounded-lg">
                <h3 class="text-cyan-500 font-medium border-b border-slate-800 pb-2 mb-2">Resumo da Esteira (Pre-Flight Check)</h3>
                <div class="font-mono text-xs text-slate-400 space-y-2">
                    <span class="block text-slate-500">>> Target: <span class="text-slate-300">ERP Gemco Financeiro</span></span>
                    <span class="block text-slate-500">>> Include Base v11: <span class="text-slate-300">${this.selectedAction === 'install_full' ? 'Sim' : 'Não'}</span></span>
                    <span class="block text-slate-500">>> Mutex Flow: <span class="text-cyan-400/80">${actionDesc}</span></span>
                    <div class="mt-3 p-2 bg-slate-900 border border-slate-800 rounded">
                        <span class="text-slate-500 block mb-1">Queue (SP/Custom):</span>
                        <span class="text-slate-300">+ ${filaDesc}</span>
                    </div>
                </div>
            </div>
        `;
    }
}