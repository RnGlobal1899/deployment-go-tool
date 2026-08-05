import { InstallSoftware } from '../../wailsjs/go/main/App';

export interface SoftwareModule {
    id: string;
    name: string;
    description: string;
    iconSvg: string;
}

export class SoftwareCard {
    private element: HTMLElement;

    constructor(data: SoftwareModule) {
        const template = document.getElementById('software-card-template') as HTMLTemplateElement;
        if (!template) {
            throw new Error("[UI Error] Template 'software-card-template' não localizado no DOM.");
        }

        // Clona o fragmento de forma isolada para performance
        const clone = template.content.cloneNode(true) as DocumentFragment;
        this.element = clone.firstElementChild as HTMLElement;

        // Injeta os dados base
        this.element.id = `card-${data.id}`;
        this.element.querySelector('.card-title')!.textContent = data.name;
        this.element.querySelector('.card-description')!.textContent = data.description;

        // Injeta o SVG recebendo tratamento para herdar a Accent Color
        const iconContainer = this.element.querySelector('.card-icon-container') as HTMLElement;
        iconContainer.innerHTML = data.iconSvg;
        
        const svg = iconContainer.querySelector('svg');
        if (svg) {
            // Força a herança de cor do pai via classes Tailwind nativas
            svg.classList.add('w-7', 'h-7', 'text-current', 'fill-current');
        }

        // Executa o motor dinâmico de Cores e Relevo
        this.applyDynamicAccent(data.name, iconContainer);

        // Injeta o Event listener de clique para disparar a instalação do software
        this.element.addEventListener('click', async () => {
            const statusSpan = this.element.querySelector('.card-status') as HTMLElement;
            
            if (statusSpan) {
                statusSpan.textContent = 'Enviando...';
                statusSpan.classList.remove('opacity-0');
                statusSpan.classList.add('opacity-100', 'animate-pulse');
            }
            
            try {
                // Invoca o método Go de forma assíncrona. O Go cuidará das Goroutines de rede 
                // e do Mutex de instalação Sequencial. A interface permanece 100% responsiva.
                await InstallSoftware(data.id);
            } catch (error) {
                console.error(`[IPC Error] Falha ao disparar módulo ${data.id}:`, error);
            } finally {
                if (statusSpan) {
                    statusSpan.classList.remove('animate-pulse');
                    statusSpan.textContent = 'Processando no Backend';
                    // Aguarda 3 segundos e oculta a flag suavemente (Dark Tech style)
                    setTimeout(() => {
                        statusSpan.classList.replace('opacity-100', 'opacity-0');
                    }, 3000);
                }
            }
        });
    }


    /**
     * Motor de Accent Colors: Injeta tipografia e sombra interativa
     * baseada no software alvo para o efeito "Dark Tech / Neon Contido".
     */
    private applyDynamicAccent(name: string, iconContainer: HTMLElement) {
        const lowerName = name.toLowerCase();
        
        // Estado visual Padrão (Fallback)
        let textColorClass = 'text-slate-500';
        let hoverShadowClass = 'hover:shadow-slate-500/30';

        // Mapeamento Estratégico de Cores
        if (lowerName.includes('kaspersky')) {
            textColorClass = 'text-emerald-500';
            hoverShadowClass = 'hover:shadow-emerald-500/30';
        } else if (lowerName.includes('wps')) {
            textColorClass = 'text-amber-500';
            hoverShadowClass = 'hover:shadow-amber-500/30';
        } else if (lowerName.includes('gemco')) {
            textColorClass = 'text-cyan-500';
            hoverShadowClass = 'hover:shadow-cyan-500/30';
        } else if (lowerName.includes('anydesk')) {
            textColorClass = 'text-red-500';
            hoverShadowClass = 'hover:shadow-red-500/30';
        } else if (lowerName.includes('ocs')) {
            textColorClass = 'text-indigo-500';
            hoverShadowClass = 'hover:shadow-indigo-500/30';
        }

        // 1. Aplica a cor ao container do ícone (o SVG reage via text-current)
        iconContainer.classList.add(textColorClass);
        
        // 2. Aplica a aura/sombra de relevo direcional no container inteiro durante o Hover
        this.element.classList.add(hoverShadowClass);
    }

    /**
     * Retorna o nó HTML encapsulado e configurado, pronto para o append na Main Area.
     */
    public render(): HTMLElement {
        return this.element;
    }
}