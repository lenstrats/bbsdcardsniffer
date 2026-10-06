export namespace device {
	
	export class Device {
	    id: string;
	    path: string;
	    node: string;
	    name: string;
	    size: number;
	    sizeHuman: string;
	    removable: boolean;
	    internal: boolean;
	    bus: string;
	    content: string;
	    mounted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.node = source["node"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.sizeHuman = source["sizeHuman"];
	        this.removable = source["removable"];
	        this.internal = source["internal"];
	        this.bus = source["bus"];
	        this.content = source["content"];
	        this.mounted = source["mounted"];
	    }
	}

}

export namespace imaging {
	
	export class Result {
	    bytes: number;
	    sha256: string;
	    fileBytes: number;
	    seconds: number;
	    path: string;
	    verified: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bytes = source["bytes"];
	        this.sha256 = source["sha256"];
	        this.fileBytes = source["fileBytes"];
	        this.seconds = source["seconds"];
	        this.path = source["path"];
	        this.verified = source["verified"];
	    }
	}

}

export namespace main {
	
	export class AssistantStatus {
	    hasKey: boolean;
	    keyFromEnv: boolean;
	    writable: boolean;
	    busy: boolean;
	    backupPath: string;
	
	    static createFrom(source: any = {}) {
	        return new AssistantStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hasKey = source["hasKey"];
	        this.keyFromEnv = source["keyFromEnv"];
	        this.writable = source["writable"];
	        this.busy = source["busy"];
	        this.backupPath = source["backupPath"];
	    }
	}
	export class CloneOptions {
	    device: string;
	    trim: boolean;
	    compress: boolean;
	    verify: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CloneOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device = source["device"];
	        this.trim = source["trim"];
	        this.compress = source["compress"];
	        this.verify = source["verify"];
	    }
	}
	export class DiskInfo {
	    path: string;
	    size: number;
	    sizeHuman: string;
	    table: string;
	    volumes: volume.Volume[];
	    writable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DiskInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.size = source["size"];
	        this.sizeHuman = source["sizeHuman"];
	        this.table = source["table"];
	        this.volumes = this.convertValues(source["volumes"], volume.Volume);
	        this.writable = source["writable"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Environment {
	    os: string;
	    elevated: boolean;
	    canUnmount: boolean;
	    autoOpen: string;
	
	    static createFrom(source: any = {}) {
	        return new Environment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.os = source["os"];
	        this.elevated = source["elevated"];
	        this.canUnmount = source["canUnmount"];
	        this.autoOpen = source["autoOpen"];
	    }
	}
	export class ExportResult {
	    dest: string;
	    files: number;
	    bytes: number;
	    skipped: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dest = source["dest"];
	        this.files = source["files"];
	        this.bytes = source["bytes"];
	        this.skipped = source["skipped"];
	    }
	}
	export class ImageInfo {
	    path: string;
	    name: string;
	    size: number;
	    sizeHuman: string;
	    format: string;
	    written: number;
	    writtenHuman: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.sizeHuman = source["sizeHuman"];
	        this.format = source["format"];
	        this.written = source["written"];
	        this.writtenHuman = source["writtenHuman"];
	    }
	}
	export class Preview {
	    data: string;
	    offset: number;
	    length: number;
	    total: number;
	    totalHuman: string;
	    isText: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.offset = source["offset"];
	        this.length = source["length"];
	        this.total = source["total"];
	        this.totalHuman = source["totalHuman"];
	        this.isText = source["isText"];
	    }
	}
	export class TargetInfo {
	    path: string;
	    node: string;
	    size: number;
	    sizeHuman: string;
	    table: string;
	    volumes: volume.Volume[];
	    empty: boolean;
	    unreadable: boolean;
	    detail?: string;
	
	    static createFrom(source: any = {}) {
	        return new TargetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.node = source["node"];
	        this.size = source["size"];
	        this.sizeHuman = source["sizeHuman"];
	        this.table = source["table"];
	        this.volumes = this.convertValues(source["volumes"], volume.Volume);
	        this.empty = source["empty"];
	        this.unreadable = source["unreadable"];
	        this.detail = source["detail"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TrimInfo {
	    dataEnd: number;
	    dataEndHuman: string;
	    total: number;
	    totalHuman: string;
	    table: string;
	    losesBackupGpt: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TrimInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataEnd = source["dataEnd"];
	        this.dataEndHuman = source["dataEndHuman"];
	        this.total = source["total"];
	        this.totalHuman = source["totalHuman"];
	        this.table = source["table"];
	        this.losesBackupGpt = source["losesBackupGpt"];
	    }
	}
	export class WriteRequest {
	    image: string;
	    device: string;
	    verify: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WriteRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.image = source["image"];
	        this.device = source["device"];
	        this.verify = source["verify"];
	    }
	}

}

export namespace volume {
	
	export class Entry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    isSymlink: boolean;
	    size: number;
	    sizeHuman: string;
	    mode: string;
	    modTime: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.isSymlink = source["isSymlink"];
	        this.size = source["size"];
	        this.sizeHuman = source["sizeHuman"];
	        this.mode = source["mode"];
	        this.modTime = source["modTime"];
	    }
	}
	export class FSInfo {
	    kind: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new FSInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.label = source["label"];
	    }
	}
	export class Volume {
	    index: number;
	    name: string;
	    uuid: string;
	    start: number;
	    size: number;
	    sizeHuman: string;
	    fs: FSInfo;
	    supported: boolean;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new Volume(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.uuid = source["uuid"];
	        this.start = source["start"];
	        this.size = source["size"];
	        this.sizeHuman = source["sizeHuman"];
	        this.fs = this.convertValues(source["fs"], FSInfo);
	        this.supported = source["supported"];
	        this.label = source["label"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

