import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

// The committed OpenAPI spec lives in the Go module's codegen tree and is
// versioned in its filename (e.g. integration-10.1.78.json). Resolve it by glob
// so a spec bump never requires editing the site.
const SPEC_DIR = join(process.cwd(), '..', 'codegen', 'openapi');

export function resolveSpecFile(): string {
  // Natural (numeric) sort so the highest version wins: a plain lexicographic
  // sort would rank integration-9.x after integration-10.x, and 10.1.9 after
  // 10.1.78 — silently selecting an older spec.
  const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' });
  const file = readdirSync(SPEC_DIR)
    .filter((f) => /^integration-.*\.json$/.test(f))
    .sort((a, b) => collator.compare(a, b))
    .at(-1);
  if (!file) throw new Error(`No integration-*.json OpenAPI spec found in ${SPEC_DIR}`);
  return join(SPEC_DIR, file);
}

export function specVersion(): string {
  const m = /integration-(.*)\.json$/.exec(resolveSpecFile());
  return m?.[1] ?? 'unknown';
}

/**
 * Load the committed UniFi OpenAPI spec and patch it for accurate rendering.
 *
 * The upstream spec ships with no security scheme and a host-less `/integration`
 * server, so the reference would otherwise omit auth and show the wrong URL. We
 * inject what go-unifi actually sends, verified against unifi/api_paths.go:
 *   ApiKeyHeader      = "X-Api-Key"
 *   integrationV1Path = "/proxy/network/integration/v1"  (paths already carry /v1)
 */
export function loadUnifiSpec(): Record<string, unknown> {
  const spec = JSON.parse(readFileSync(resolveSpecFile(), 'utf8')) as Record<string, any>;

  spec.components ??= {};
  spec.components.securitySchemes = {
    ApiKey: {
      type: 'apiKey',
      in: 'header',
      name: 'X-Api-Key',
      description:
        'UniFi API key. Create one under Control Plane → Admins & Users → your admin → Create API Key (requires controller 9.0.114+).',
    },
  };
  spec.security = [{ ApiKey: [] }];
  spec.servers = [
    {
      url: 'https://{controller}/proxy/network/integration',
      description: 'UniFi OS controller — Network application Official API',
      variables: {
        controller: {
          default: 'unifi.example.com',
          description: 'Your UniFi controller hostname or IP address',
        },
      },
    },
  ];

  return spec;
}
