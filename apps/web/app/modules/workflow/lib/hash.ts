/**
 * Canonical JSON serialization with sorted keys.
 * Ensures deterministic output regardless of key order.
 */
export function canonicalJson(obj: unknown): string {
	if (obj === null || obj === undefined) {
		return JSON.stringify(obj);
	}

	if (typeof obj !== "object") {
		return JSON.stringify(obj);
	}

	if (Array.isArray(obj)) {
		return `[${obj.map((item) => canonicalJson(item)).join(",")}]`;
	}

	// Object: sort keys and recursively canonicalize values
	const sortedKeys = Object.keys(obj).sort();
	const entries = sortedKeys.map((key) => {
		const value = (obj as Record<string, unknown>)[key];
		return `${JSON.stringify(key)}:${canonicalJson(value)}`;
	});

	return `{${entries.join(",")}}`;
}

/**
 * Compute SHA256 hash of input string using Web Crypto API.
 * Returns base64url-encoded hash (URL-safe, no padding).
 */
export async function sha256(input: string): Promise<string> {
	const encoder = new TextEncoder();
	const data = encoder.encode(input);
	const hashBuffer = await crypto.subtle.digest("SHA-256", data);

	// Convert ArrayBuffer to base64url
	const hashArray = Array.from(new Uint8Array(hashBuffer));
	const base64 = btoa(String.fromCharCode(...hashArray));

	// Convert to base64url (URL-safe, no padding)
	return base64.replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

/**
 * Compute hash of input string.
 * Alias for sha256 for consistency.
 */
export async function computeHash(input: string): Promise<string> {
	return sha256(input);
}
