import { createOpenAPI } from 'fumadocs-openapi/server';
import { loadUnifiSpec } from './spec';

// Single in-memory schema, patched at build time (no generated files on disk,
// no version baked into committed source). `staticSource` reads this to produce
// the virtual API-reference pages; the spec is fully resolved at build for a
// static export.
export const openapi = createOpenAPI({
  input: {
    'unifi-network-api': () => loadUnifiSpec() as never,
  },
});
