export const appName = 'go-unifi';

// GitHub Pages serves this as a project site under /go-unifi. CI sets the base
// path via PAGES_BASE_PATH, which next.config.mjs re-exports as
// NEXT_PUBLIC_BASE_PATH; local builds leave it empty.
export const basePath = process.env.NEXT_PUBLIC_BASE_PATH ?? '';

// Prefix a public asset (e.g. /logo.png) with the deployment base path.
// next/image does NOT prepend basePath to `src` when images are unoptimized
// (required by `output: 'export'`), so static images 404 under /go-unifi unless
// we build the URL ourselves.
export function asset(path: string): string {
  return `${basePath}${path}`;
}

export const docsRoute = '/docs';
export const docsImageRoute = '/og/docs';
export const docsContentRoute = '/llms.mdx/docs';

export const gitConfig = {
  user: 'filipowm',
  repo: 'go-unifi',
  branch: 'main',
};

export const goModule = 'github.com/filipowm/go-unifi/v2';
export const pkgGoDev = `https://pkg.go.dev/${goModule}`;
export const repoUrl = `https://github.com/${gitConfig.user}/${gitConfig.repo}`;
