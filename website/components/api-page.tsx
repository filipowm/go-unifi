'use client';
import { createOpenAPIPage } from 'fumadocs-openapi/ui';
import { goCrosswalk } from '@/lib/go-crosswalk';

// The full reference (parameters, schemas, responses, code samples) is
// pre-rendered statically at build time. The interactive "Send" playground is
// enabled, but note it fires direct from the browser: reaching a private,
// self-signed UniFi controller may be blocked by CORS/TLS.
export const OpenAPIPage = createOpenAPIPage({
  playground: { enabled: true },
  showResponseSchema: true,
  // Add a "Go (go-unifi)" sample showing the fluent Official-SDK call for each
  // operation, mapped from operationId. Keeps the default curl/JS samples too.
  generateCodeSamples({ operation }) {
    const id = (operation as { operationId?: string }).operationId;
    const call = id ? goCrosswalk[id] : undefined;
    if (!call) return [];
    return [
      {
        lang: 'go',
        label: 'Go (go-unifi)',
        source: `// ctx context.Context, c unifi.Client, siteID uuid.UUID\n// import "github.com/filipowm/go-unifi/v2/unifi" and "github.com/filipowm/go-unifi/v2/unifi/official"\n${call}`,
      },
    ];
  },
});
