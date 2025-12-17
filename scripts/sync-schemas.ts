#!/usr/bin/env bun
/// <reference types="node" />
/// <reference path="./bun-types.d.ts" />
/**
 * Sync schemas and OpenAPI files from spec/ to apps/web/public/
 * 
 * This script ensures runtime assets are kept in sync with canonical sources.
 */

import { readdir, readFile, writeFile, mkdir } from "fs/promises";
import { existsSync } from "fs";
import { join, dirname, resolve } from "path";

const ROOT = resolve((import.meta as ImportMeta & { dir: string }).dir, "..");
const SPEC_ROOT = join(ROOT, "spec");
const PUBLIC_ROOT = join(ROOT, "apps/web/public");

const SCHEMA_ORIGIN = process.env.PUBLIC_SCHEMA_ORIGIN || "http://localhost:5173";

async function ensureDir(path: string) {
	if (!existsSync(path)) {
		await mkdir(path, { recursive: true });
	}
}

async function syncFile(
	sourcePath: string,
	destPath: string,
	transform?: (content: string) => string
) {
	const content = await readFile(sourcePath, "utf-8");
	const transformed = transform ? transform(content) : content;
	await ensureDir(dirname(destPath));
	await writeFile(destPath, transformed, "utf-8");
	console.log(`✓ Synced ${sourcePath} → ${destPath}`);
}

function rewriteSchemaOrigin(content: string): string {
	// Rewrite $id host from https://agentlegible.dev to PUBLIC_SCHEMA_ORIGIN
	return content.replace(
		/https:\/\/agentlegible\.dev/g,
		SCHEMA_ORIGIN
	);
}

async function syncSchemas() {
	console.log("Syncing schemas...\n");

	// Sync meta schema
	await syncFile(
		join(SPEC_ROOT, "jsonschema/meta/draft2020-12.schema.json"),
		join(PUBLIC_ROOT, "schemas/meta/draft2020-12.schema.json")
	);

	// Sync SMALL v1 schemas
	const schemaFiles = [
		"schema.schema.json",
		"manifest.schema.json",
		"artifact.schema.json",
		"lineage.schema.json",
		"lifecycle.schema.json",
	];

	for (const file of schemaFiles) {
		await syncFile(
			join(SPEC_ROOT, "jsonschema/small/v1", file),
			join(PUBLIC_ROOT, "schemas/small/v1", file),
			rewriteSchemaOrigin
		);
	}
}

async function syncOpenAPI() {
	console.log("\nSyncing OpenAPI files...\n");

	const openapiFiles = await readdir(join(SPEC_ROOT, "openapi"));

	for (const file of openapiFiles) {
		if (file.endsWith(".yaml") || file.endsWith(".yml")) {
			await syncFile(
				join(SPEC_ROOT, "openapi", file),
				join(PUBLIC_ROOT, "openapi", file)
			);
		}
	}
}

async function main() {
	console.log(`Schema origin: ${SCHEMA_ORIGIN}\n`);

	try {
		await syncSchemas();
		await syncOpenAPI();
		console.log("\n✓ Sync complete!");
	} catch (error) {
		console.error("\n✗ Sync failed:", error);
		process.exit(1);
	}
}

main();
