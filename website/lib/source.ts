import { createElement } from 'react';
import { docs } from 'collections/server';
import { loader } from 'fumadocs-core/source';
import { icons } from 'lucide-react';
import { openapi } from './openapi';
import { docsContentRoute, docsImageRoute, docsRoute } from './shared';

// Multi-source loader: hand-written MDX docs + virtual OpenAPI pages (one per
// operation, grouped by tag under `reference/api`). Both merge into one page
// tree under /docs. See https://fumadocs.dev/docs/ui/openapi
export const source = loader(
  {
    docs: docs.toFumadocsSource(),
    openapi: await openapi.staticSource({
      groupBy: 'tag',
      baseDir: 'reference/api',
    }),
  },
  {
    baseUrl: docsRoute,
    plugins: [openapi.loaderPlugin()],
    icon(icon) {
      if (icon && icon in icons) {
        return createElement(icons[icon as keyof typeof icons]);
      }
    },
  },
);

export type DocsPage = (typeof source)['$inferPage'];

export function getPageImage(page: DocsPage) {
  const segments = [...page.slugs, 'image.png'];
  return {
    segments,
    url: `${docsImageRoute}/${segments.join('/')}`,
  };
}

export function getPageMarkdownUrl(page: DocsPage) {
  const segments = [...page.slugs, 'content.md'];
  return {
    segments,
    url: `${docsContentRoute}/${segments.join('/')}`,
  };
}

export async function getLLMText(page: DocsPage) {
  const heading = `# ${page.data.title} (${page.url})`;
  // OpenAPI pages are generated and have no processed markdown body.
  if (page.type === 'openapi') {
    return `${heading}\n\n${page.data.description ?? ''}`;
  }
  const processed = await page.data.getText('processed');
  return `${heading}\n\n${processed}`;
}
