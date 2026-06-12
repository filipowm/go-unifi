'use client';
import { createOpenAPIPage } from 'fumadocs-openapi/ui';

// The full reference (parameters, schemas, responses, code samples) is
// pre-rendered statically at build time. The interactive "Send" playground is
// enabled, but note it fires direct from the browser: reaching a private,
// self-signed UniFi controller may be blocked by CORS/TLS.
export const OpenAPIPage = createOpenAPIPage({
  playground: { enabled: true },
  showResponseSchema: true,
});
