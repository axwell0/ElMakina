import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { compile } from "json-schema-to-typescript";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const repoRoot = path.resolve(__dirname, "..");

const schemaPath = path.resolve(repoRoot, "..", "shared", "schemas", "envelope.json");

if (!fs.existsSync(schemaPath)) {
  throw new Error(`Could not find schema at ${schemaPath}`);
}

const schema = JSON.parse(fs.readFileSync(schemaPath, "utf8"));
const output = await compile(schema, "ElMakinaWebSocketEnvelope", {
  bannerComment:
    "/* eslint-disable */\n" +
    "/**\n" +
    " * This file is auto-generated from the backend WebSocket schema.\n" +
    " * Source: shared/schemas/envelope.json\n" +
    " * Do not edit by hand.\n" +
    " */",
  style: { singleQuote: true },
});

const outDir = path.join(repoRoot, "src", "network");
const outFile = path.join(outDir, "ws-contract.ts");
const schemaOutFile = path.join(outDir, "ws-schema.json");

fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(outFile, output);
fs.writeFileSync(schemaOutFile, JSON.stringify(schema, null, 2));

console.log(`Generated ${path.relative(repoRoot, outFile)} from ${schemaPath}`);
console.log(`Generated ${path.relative(repoRoot, schemaOutFile)} from ${schemaPath}`);
