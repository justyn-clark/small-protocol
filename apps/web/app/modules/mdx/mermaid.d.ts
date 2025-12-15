declare module "mermaid" {
	interface MermaidConfig {
		startOnLoad?: boolean;
		theme?: string;
		securityLevel?: string;
		fontFamily?: string;
	}

	interface Mermaid {
		initialize(config: MermaidConfig): void;
		render(id: string, definition: string): Promise<{ svg: string }>;
	}

	const mermaid: Mermaid;
	export default mermaid;
}
