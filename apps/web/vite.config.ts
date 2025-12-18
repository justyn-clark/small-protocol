import path from "node:path";
import { reactRouter } from "@react-router/dev/vite";
import mdx from "@mdx-js/rollup";
import rehypeHighlight from 'rehype-highlight';
import type { Plugin } from "vite";
import { defineConfig } from "vite";

function mdxHmrPlugin(): Plugin {
	return {
		name: "mdx-hmr",
		configureServer(server) {
			const mdxDir = path.resolve(__dirname, "./app/modules/docs/content");
			server.watcher.add(mdxDir);
			server.watcher.on("change", (file) => {
				if (file.endsWith(".mdx")) {
					server.ws.send({
						type: "full-reload",
					});
				}
			});
		},
	};
}

const options = {
	// See https://mdxjs.com/advanced/plugins
	remarkPlugins: [
		// E.g. `remark-frontmatter`
	],
	rehypePlugins: [rehypeHighlight],
};


export default defineConfig({
	plugins: [mdx({
		rehypePlugins: [...options.rehypePlugins],
	}),
	mdxHmrPlugin(),
	reactRouter()
	],
	resolve: {
		alias: {
			"~": path.resolve(__dirname, "./app"),
		},
	},
});
