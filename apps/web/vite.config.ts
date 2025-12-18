import path from "node:path";
import { reactRouter } from "@react-router/dev/vite";
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

export default defineConfig({
	plugins: [reactRouter(), mdxHmrPlugin()],
	resolve: {
		alias: {
			"~": path.resolve(__dirname, "./app"),
		},
	},
});
