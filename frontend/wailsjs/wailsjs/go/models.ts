export namespace main {
	
	export class QuotaInfo {
	    geniusBot: number;
	    credits: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new QuotaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.geniusBot = source["geniusBot"];
	        this.credits = source["credits"];
	        this.error = source["error"];
	    }
	}
	export class ServiceStatus {
	    isRunning: boolean;
	    message: string;
	    address?: string;
	    apiKey?: string;
	
	    static createFrom(source: any = {}) {
	        return new ServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRunning = source["isRunning"];
	        this.message = source["message"];
	        this.address = source["address"];
	        this.apiKey = source["apiKey"];
	    }
	}
	export class WailsTestResult {
	    endpoint: string;
	    url: string;
	    requestData: string;
	    responseData: string;
	    statusCode: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new WailsTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.endpoint = source["endpoint"];
	        this.url = source["url"];
	        this.requestData = source["requestData"];
	        this.responseData = source["responseData"];
	        this.statusCode = source["statusCode"];
	        this.error = source["error"];
	    }
	}

}

