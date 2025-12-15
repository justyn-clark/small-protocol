export interface Manifest {
	artifact: string;
	schema: string;
	version: number;
	metadata?: Record<string, unknown>;
}

export interface Lineage {
	artifact: string;
	derivedFrom: string;
	generatedAt: string;
}

export type Lifecycle = Array<{
	type: string;
	at: string;
}>;
