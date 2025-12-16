import { SMALL_PROTOCOL } from "~/modules/protocol/version";
import { SMALL_PRIMITIVES } from "~/modules/protocol/small.protocol";
import type { Frontmatter } from "~/modules/mdx/mdx-runtime.server";

export function validateProtocolDoc(
  frontmatter: Frontmatter,
  filePath: string,
): void {
  // If no protocol field, skip validation (not a protocol doc)
  if (!frontmatter.protocol) {
    return;
  }

  // Validate protocol name
  if (frontmatter.protocol !== "SMALL") {
    throw new Error(
      `Invalid protocol in ${filePath}: expected "SMALL", got "${frontmatter.protocol}"`,
    );
  }

  // Validate version matches protocol version
  if (frontmatter.version !== SMALL_PROTOCOL.version) {
    throw new Error(
      `Protocol version mismatch in ${filePath}: expected "${SMALL_PROTOCOL.version}", got "${frontmatter.version}"`,
    );
  }

  // Validate primitive if present
  if (frontmatter.primitive) {
    const primitive = String(frontmatter.primitive);
    if (!SMALL_PRIMITIVES.includes(primitive as (typeof SMALL_PRIMITIVES)[number])) {
      throw new Error(
        `Invalid primitive in ${filePath}: "${primitive}" is not a valid SMALL primitive. Valid primitives: ${SMALL_PRIMITIVES.join(", ")}`,
      );
    }
  }
}

