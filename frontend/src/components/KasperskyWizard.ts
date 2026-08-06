import * as AppIPC from '../../wailsjs/go/main/App';

const CheckKasperskyStatus = (AppIPC as any).CheckKasperskyStatus || (async () => false);
const ExecuteKasperskyWizard = (AppIPC as any).ExecuteKasperskyWizard || (async () => {});

export class KasperskyWizard {
    private root: HTMLElement;
    private content: HTMLElement;
    private btnNext: HTMLButtonElement;
    
    // Controle de Máquina de Estados
    private step: number = 1;
    private targetMenu: 'agent' | 'endpoint' | 'both' = 'both';
    
    // Flags de Instalação (Buscadas no Go)
    private isAgentInstalled: boolean = false;
    private isEndpointInstalled: boolean = false;

    // Respostas finais a serem enviadas ao backend
    private agentFinalAction: string = "";
    private agentFinalPayload: string = "";
    private kesFinalAction: string = "";
    private kesFinalPayload: string = "";

    constructor() {
        this.root = document.getElementById('modal-root') as HTMLElement;
        const template = document.getElementById('kaspersky-wizard-template') as HTMLTemplateElement;
        
        this.root.innerHTML = ''; // Limpa execuções anteriores
        this.root.appendChild(template.content.cloneNode(true));
        
        this.content = this.root.querySelector('.wizard-content') as HTMLElement;
        this.btnNext = this.root.querySelector('.btn-next') as HTMLButtonElement;
        const btnCancel = this.root.querySelector('.btn-cancel') as HTMLButtonElement;

        btnCancel.addEventListener('click', () => this.unmount());
        this.btnNext.addEventListener('click', () => this.handleNextStep());
    }

    public mount() {
        this.root.classList.remove('hidden');
        this.root.classList.add('flex');
        this.renderStep1();
    }

    private unmount() {
        this.root.classList.add('hidden');
        this.root.classList.remove('flex');
        this.root.innerHTML = '';
    }

    private async handleNextStep() {
        if (this.step === 1) {
            const select = this.content.querySelector('#target-select') as HTMLSelectElement;
            this.targetMenu = select.value as 'agent' | 'endpoint' | 'both';
            this.btnNext.textContent = "Verificando...";
            this.btnNext.disabled = true;

            // Busca os status reais no Windows via Go
            if (this.targetMenu === 'agent' || this.targetMenu === 'both') {
                this.isAgentInstalled = await CheckKasperskyStatus('agent');
            }
            if (this.targetMenu === 'endpoint' || this.targetMenu === 'both') {
                this.isEndpointInstalled = await CheckKasperskyStatus('endpoint');
            }

            this.btnNext.disabled = false;
            this.step = (this.targetMenu === 'endpoint') ? 3 : 2; // Pula o step do agente se escolheu só endpoint
            this.renderCurrentStep();
            return;
        }

        if (this.step === 2) {
            const action = (this.content.querySelector('#agent-action') as HTMLSelectElement).value;
            const ipInput = this.content.querySelector('#agent-ip') as HTMLInputElement;
            
            this.agentFinalAction = action;
            this.agentFinalPayload = ipInput ? ipInput.value : "";

            this.step = (this.targetMenu === 'agent') ? 4 : 3; // Vai pro fim se era só agente, senão vai pro endpoint
            this.renderCurrentStep();
            return;
        }

        if (this.step === 3) {
            const action = (this.content.querySelector('#kes-action') as HTMLSelectElement).value;
            const licInput = this.content.querySelector('#kes-license') as HTMLInputElement;
            
            this.kesFinalAction = action;
            this.kesFinalPayload = licInput ? licInput.value : "";

            this.step = 4;
            this.renderCurrentStep();
            return;
        }

        if (this.step === 3) {
            if (this.isEndpointInstalled) {
                const action = (this.content.querySelector('#kes-action') as HTMLSelectElement).value;
                const licInput = this.content.querySelector('#kes-license') as HTMLInputElement;
                
                this.kesFinalAction = action;
                this.kesFinalPayload = licInput ? licInput.value : "";
            } else {
                this.kesFinalAction = "install";
            }

            this.step = 4;
            this.renderCurrentStep();
            return;
        }

        if (this.step === 4) {
            // Executa a esteira final
            this.btnNext.textContent = "Disparando...";
            await ExecuteKasperskyWizard(this.agentFinalAction, this.agentFinalPayload, this.kesFinalAction, this.kesFinalPayload);
            this.unmount();
        }
    }

    private renderCurrentStep() {
        if (this.step === 2) this.renderStep2();
        else if (this.step === 3) this.renderStep3();
        else if (this.step === 4) this.renderStep4();
    }

    private renderStep1() {
        this.btnNext.textContent = "Avançar";
        this.content.innerHTML = `
            <div class="space-y-2">
                <label class="block text-slate-400 font-medium">O que deseja gerenciar?</label>
                <select id="target-select" class="w-full bg-slate-950 border border-slate-700 text-slate-200 rounded-lg p-2.5 outline-none focus:border-emerald-500 transition-colors">
                    <option value="both">Suíte Completa (Agente + Endpoint)</option>
                    <option value="agent">Apenas Network Agent</option>
                    <option value="endpoint">Apenas Endpoint Security</option>
                </select>
            </div>
        `;
    }

    private renderStep2() {
        this.btnNext.textContent = "Avançar";

        const statusBadge = this.isAgentInstalled 
            ? `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-emerald-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                  <span>Network Agent Instalado</span>
               </div>`
            : `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-amber-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-amber-500 animate-pulse"></span>
                  <span>Network Agent Ausente</span>
               </div>`;

        const options = this.isAgentInstalled
            ? `<option value="skip">Seguir em Frente (Ignorar)</option>
               <option value="repoint">Reapontar Agente</option>
               <option value="reinstall">Reinstalar (Clean Install)</option>
               <option value="uninstall">Apenas Desinstalar</option>`
            : `<option value="install">Instalação Limpa</option>
               <option value="skip">Ignorar Instalação</option>`;

        this.content.innerHTML = `
            <div class="space-y-4">
                ${statusBadge}
                <div class="space-y-2">
                    <label class="block text-slate-400 font-medium">Ação no Agente:</label>
                    <select id="agent-action" class="w-full bg-slate-950 border border-slate-700 text-slate-200 rounded-lg p-2.5 outline-none focus:border-emerald-500 transition-colors">
                        ${options}
                    </select>
                </div>
                
                <div id="agent-ip-container" class="space-y-2 animate-fade-in-up">
                    <label class="block text-slate-400 font-medium">Target IP (Administration Server):</label>
                    <input type="text" id="agent-ip" placeholder="Ex: 192.168.0.100" class="w-full bg-slate-950 border border-slate-700 text-slate-200 rounded-lg p-2.5 outline-none focus:border-emerald-500 transition-colors" />
                </div>
            </div>
        `;

        const actionSelect = this.content.querySelector('#agent-action') as HTMLSelectElement;
        const ipContainer = this.content.querySelector('#agent-ip-container') as HTMLElement;
        
        const toggleIP = () => {
            if (actionSelect.value === 'repoint' || actionSelect.value === 'reinstall' || actionSelect.value === 'install') {
                ipContainer.classList.remove('hidden');
            } else {
                ipContainer.classList.add('hidden');
            }
        };

        toggleIP();
        actionSelect.addEventListener('change', toggleIP);
    }

   private renderStep3() {
        this.btnNext.textContent = "Avançar";

        const statusBadge = this.isEndpointInstalled 
            ? `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-emerald-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
                  <span>Endpoint Security Instalado</span>
               </div>`
            : `<div class="p-3 bg-slate-950 border border-slate-800 rounded-lg text-amber-500 font-mono text-xs flex items-center space-x-2">
                  <span class="w-2 h-2 rounded-full bg-amber-500 animate-pulse"></span>
                  <span>Endpoint Security Ausente</span>
               </div>`;

        const options = this.isEndpointInstalled
            ? `<option value="skip">Seguir em Frente (Ignorar)</option>
               <option value="activate">Ativar Licença via AVP</option>
               <option value="reinstall">Reinstalar (Clean Install)</option>
               <option value="uninstall">Desinstalar</option>`
            : `<option value="install">Instalação Limpa</option>
               <option value="skip">Ignorar Instalação</option>`;

        this.content.innerHTML = `
            <div class="space-y-4">
                ${statusBadge}
                <div class="space-y-2">
                    <label class="block text-slate-400 font-medium">Ação no Endpoint:</label>
                    <select id="kes-action" class="w-full bg-slate-950 border border-slate-700 text-slate-200 rounded-lg p-2.5 outline-none focus:border-emerald-500 transition-colors">
                        ${options}
                    </select>
                </div>
                
                <div id="kes-lic-container" class="space-y-2 animate-fade-in-up">
                    <label class="block text-slate-400 font-medium">Licença Corporativa (Opcional):</label>
                    <input type="text" id="kes-license" placeholder="XXXXX-XXXXX-XXXXX-XXXXX" class="w-full bg-slate-950 border border-slate-700 text-slate-200 rounded-lg p-2.5 outline-none focus:border-emerald-500 transition-colors font-mono uppercase" />
                </div>
            </div>
        `;

        const actionSelect = this.content.querySelector('#kes-action') as HTMLSelectElement;
        const licContainer = this.content.querySelector('#kes-lic-container') as HTMLElement;
        
        const toggleLic = () => {
            if (actionSelect.value === 'activate' || actionSelect.value === 'install' || actionSelect.value === 'reinstall') {
                licContainer.classList.remove('hidden');
            } else {
                licContainer.classList.add('hidden');
            }
        };

        toggleLic(); // Força estado inicial correto
        actionSelect.addEventListener('change', toggleLic);
    }

    private renderStep4() {
        this.btnNext.textContent = "Confirmar Execução";
        this.content.innerHTML = `
            <div class="space-y-3 bg-slate-950 border border-slate-800 p-4 rounded-lg">
                <h3 class="text-emerald-500 font-medium border-b border-slate-800 pb-2 mb-2">Resumo da Esteira (Pre-Flight Check)</h3>
                <div class="font-mono text-xs text-slate-400">
                    <span class="block text-slate-500">>> Agente Action: <span class="text-slate-300">${this.agentFinalAction || 'N/A'}</span></span>
                    <span class="block text-slate-500">>> Agente Alvo IP: <span class="text-slate-300">${this.agentFinalPayload || 'N/A'}</span></span>
                    <span class="block mt-2 text-slate-500">>> Endpoint Action: <span class="text-slate-300">${this.kesFinalAction || 'N/A'}</span></span>
                    <span class="block text-slate-500">>> Endpoint Licença: <span class="text-slate-300">${this.kesFinalPayload ? '******-INJETADA' : 'N/A'}</span></span>
                </div>
            </div>
        `;
    }
}