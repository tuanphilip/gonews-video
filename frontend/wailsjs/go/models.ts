export namespace config {
	
	export class Config {
	    nineRouterUrl: string;
	    nineRouterKey: string;
	    nineRouterModel: string;
	    outputDir: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nineRouterUrl = source["nineRouterUrl"];
	        this.nineRouterKey = source["nineRouterKey"];
	        this.nineRouterModel = source["nineRouterModel"];
	        this.outputDir = source["outputDir"];
	    }
	}

}

export namespace main {
	
	export class ArticlePreview {
	    taskId: string;
	    url: string;
	    title: string;
	    headline1: string;
	    headline2: string;
	    headline: string;
	    fullScript: string;
	    sentences: string[];
	    images: string[];
	
	    static createFrom(source: any = {}) {
	        return new ArticlePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.headline1 = source["headline1"];
	        this.headline2 = source["headline2"];
	        this.headline = source["headline"];
	        this.fullScript = source["fullScript"];
	        this.sentences = source["sentences"];
	        this.images = source["images"];
	    }
	}
	export class ProcessProgress {
	    step: string;
	    percent: number;
	    log: string;
	    isComplete: boolean;
	    error?: string;
	    videoPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.step = source["step"];
	        this.percent = source["percent"];
	        this.log = source["log"];
	        this.isComplete = source["isComplete"];
	        this.error = source["error"];
	        this.videoPath = source["videoPath"];
	    }
	}
	export class RenderRequest {
	    taskId: string;
	    headline1: string;
	    headline2: string;
	    headline: string;
	    footerColor: string;
	    tagText: string;
	    script: string;
	    images: string[];
	    voiceName: string;
	    ratio: string;
	
	    static createFrom(source: any = {}) {
	        return new RenderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.headline1 = source["headline1"];
	        this.headline2 = source["headline2"];
	        this.headline = source["headline"];
	        this.footerColor = source["footerColor"];
	        this.tagText = source["tagText"];
	        this.script = source["script"];
	        this.images = source["images"];
	        this.voiceName = source["voiceName"];
	        this.ratio = source["ratio"];
	    }
	}

}

