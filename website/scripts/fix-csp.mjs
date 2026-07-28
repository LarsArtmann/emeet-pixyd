import { readdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { createHash } from "node:crypto";

const DIST = "dist";

const STYLE_HASH_RE = / style-src 'self' 'unsafe-inline'(?: 'sha256-[^']+')+/g;

const INLINE_SCRIPT_RE =
  /<script(?![^>]*\btype\s*=\s*["']module["'])(?![^>]*\bsrc\s*=)([^>]*)>([\s\S]*?)<\/script>/g;

async function findHtmlFiles(dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await findHtmlFiles(full)));
    } else if (entry.name.endsWith(".html")) {
      files.push(full);
    }
  }
  return files;
}

function sha256Base64(content) {
  return "sha256-" + createHash("sha256").update(content, "utf-8").digest("base64");
}

async function main() {
  const files = await findHtmlFiles(DIST);
  let patched = 0;

  for (const file of files) {
    const html = await readFile(file, "utf-8");
    let fixed = html.replace(STYLE_HASH_RE, " style-src 'self' 'unsafe-inline'");

    const missingHashes = new Set();
    for (const [, , body] of fixed.matchAll(INLINE_SCRIPT_RE)) {
      const trimmed = body.trim();
      if (!trimmed) continue;
      const hash = sha256Base64(trimmed);
      missingHashes.add(`'${hash}'`);
    }

    if (missingHashes.size > 0) {
      const hashList = [...missingHashes].join(" ");
      fixed = fixed.replace(/script-src ('self'(?: 'sha256-[^']+')*)/, `script-src $1 ${hashList}`);
    }

    if (fixed !== html) {
      await writeFile(file, fixed);
      patched++;
    }
  }

  console.log(`CSP fix: patched ${patched}/${files.length} HTML files`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
