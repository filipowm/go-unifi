'use client';
import { createOpenAPIPage } from 'fumadocs-openapi/ui';

// The interactive "Send" playground is intentionally disabled: a browser (or a
// cloud edge) cannot reach a user's private, self-signed UniFi controller, so a
// live try-it would always fail CORS/TLS. The full reference (parameters,
// schemas, responses, code samples) is pre-rendered statically.
export const OpenAPIPage = createOpenAPIPage({
  playground: { enabled: false },
  showResponseSchema: true,
});
