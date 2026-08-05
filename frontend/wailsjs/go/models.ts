export namespace main {
	
	export class UIModule {
	    id: string;
	    name: string;
	    description: string;
	    iconSvg: string;
	
	    static createFrom(source: any = {}) {
	        return new UIModule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.iconSvg = source["iconSvg"];
	    }
	}

}

