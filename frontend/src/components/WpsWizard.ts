import * as AppIPC from '../../wailsjs/go/main/App';

const CheckWpsStatus = (AppIPC as any).CheckWpsStatus || (async () => false);
const ExecuteWpsWizard = (AppIPC as any).ExecuteWpsWizard || (async () => {});

export class WpsWizard {
    private root: HTMLElement;
    private content!: HTMLElement;
    private btnNext!: HTMLButtonElement;
    
    private step: number = 1;
    private isInstalled: boolean = false;
    private selectedAction: string = "";

    constructor() {
        this.root = document.getElementById('modal-root') as HTMLElement;
        const template = document.getElementById('wps-wizard-template') as HTMLTemplateElement;
        
        if (!template) {
            console.error("[UI Error] Template 'wps-wizard-template' ausente.");
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
        
        this.btnNext.textContent = "Lendo sistema...";
        this.btnNext.disabled = true;

        // Bate no backend (Win32) para ver se as pastas e o uninst.exe existem
        this.isInstalled = await CheckWpsStatus();

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
            const select = this.content.querySelector('#wps-action') as HTMLSelectElement;
            this.selectedAction = select.value;

            if (this.selectedAction === 'skip') {
                this.unmount();
                return;
            }

            this.step = 2;
            this.renderStep2();
            return;
        }

        if (this.step === 2) {
            this.btnNext.textContent = "Disparando API Win32...";
            this.btnNext.disabled = true;
            await ExecuteWpsWizard(this.selectedAction);
            this.unmount();
        }
    }

    private renderStep1() {
        this.btnNext.textContent = "Avançar";

        const statusBadge = this.isInstalled 
            ? `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-amber-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-amber-500 animate-pulse"></span>
                  <span>Instalação Local Detectada</span>
               </div>`
            : `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-slate-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-slate-500"></span>
                  <span>WPS Ausente na Máquina</span>
               </div>`;

        const options = this.isInstalled
            ? `<option value="skip">Não fazer nada (Abortar)</option>
               <option value="reinstall">Clean Install (Remover + Baixar + Instalar)</option>
               <option value="uninstall">Desinstalar Silenciosamente</option>`
            : `<option value="install">Download & Instalação Silenciosa</option>
               <option value="skip">Cancelar Ação</option>`;

        this.content.innerHTML = `
            <div class="space-y-4">
                ${statusBadge}
                <div class="space-y-2">
                    <label class="block text-slate-400 font-medium">Orquestração Disponível:</label>
                    <select id="wps-action" class="w-full bg-slate-950 border border-slate-700 text-slate-200 rounded-lg p-2.5 outline-none focus:border-amber-500 transition-colors">
                        ${options}
                    </select>
                </div>
                <p class="text-xs text-slate-500 font-mono border-l-2 border-slate-700 pl-2 mt-2">
                    Operações Clean Install executam busca agressiva por resíduos no 'ProgramFiles' e 'AppData'.
                </p>
            </div>
        `;
    }

    private renderStep2() {
        this.btnNext.textContent = "Confirmar Execução";
        
        let actionDesc = "N/A";
        if (this.selectedAction === 'install') actionDesc = "Download Assíncrono -> Instalação Sequencial Win32";
        if (this.selectedAction === 'reinstall') actionDesc = "Remoção Win32 -> Wait -> Download -> Nova Instalação";
        if (this.selectedAction === 'uninstall') actionDesc = "Erradicação Isolada via uninst.exe /S";

        this.content.innerHTML = `
            <div class="space-y-3 bg-slate-950 border border-slate-800 p-4 rounded-lg">
                <h3 class="text-amber-500 font-medium border-b border-slate-800 pb-2 mb-2">Resumo da Esteira (Pre-Flight Check)</h3>
                <div class="font-mono text-xs text-slate-400">
                    <span class="block text-slate-500">>> Alvo: <span class="text-slate-300">WPS Office</span></span>
                    <span class="block text-slate-500 mt-2">>> Mutex Flow: <span class="text-amber-300/80">${actionDesc}</span></span>
                </div>
            </div>
        `;
    }
}