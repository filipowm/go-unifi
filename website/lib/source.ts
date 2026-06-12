import { createElement } from 'react';
import { docs } from 'collections/server';
import { llms, loader } from 'fumadocs-core/source';
import { icons } from 'lucide-react';
import { openapi } from './openapi';
import { docsContentRoute, docsImageRoute, docsRoute } from './shared';

// Multi-source loader: hand-written MDX docs + virtual OpenAPI pages (one per
// operation, grouped by tag under `reference/official-api/api`). Both merge
// into one page tree under /docs. See https://fumadocs.dev/docs/ui/openapi
export const source = loader(
  {
    docs: docs.toFumadocsSource(),
    openapi: await openapi.staticSource({
      groupBy: 'tag',
      baseDir: 'reference/official-api/api',
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

// Reduce a (possibly multi-line, HTML-laden) description to a single clean
// summary line: keep only the text before the first newline — which drops the
// generated `<details>Filterable properties…</details>` block on OpenAPI
// operation pages — then strip any HTML/JSX tags and collapse whitespace.
function cleanDescription(desc?: string): string {
  if (!desc) return '';
  const firstLine = desc.split('\n', 1)[0];
  return firstLine
    .replace(/<[^>]*>/g, '')
    .replace(/\s+/g, ' ')
    .trim();
}

// Remove MDX/JSX component tags from processed markdown while preserving the
// inner text/markdown between them. Conservative: only Capitalized component
// names (Cards, Card, Callout, TypeTable, Tabs, Steps, Accordions, Files, …)
// plus the lowercase `include` directive. Collapses 3+ blank lines to 2.
function stripMdxComponents(md: string): string {
  return md
    .replace(/<\/?[A-Z][A-Za-z0-9]*(?:\s[^>]*?)?\/?>/g, '')
    .replace(/<\/?include(?:\s[^>]*?)?\/?>/g, '')
    .replace(/\n{3,}/g, '\n\n');
}

export async function getLLMText(page: DocsPage) {
  const heading = `# ${page.data.title} (${page.url})`;
  // OpenAPI pages are generated and have no processed markdown body.
  if (page.type === 'openapi') {
    return `${heading}\n\n${cleanDescription(page.data.description)}`;
  }
  const processed = await page.data.getText('processed');
  return `${heading}\n\n${stripMdxComponents(processed)}`;
}

// Build the /llms.txt index. fumadocs-core's `llms()` accepts a `LLMsConfig`
// with a `renderDescription` hook, so we reuse its page-tree walker (correct
// folder/separator nesting + indentation) and only clean each description to a
// single line — keeping the generated `<details>` HTML and any MDX components
// out of the index.
export function buildLLMSIndex(): string {
  return llms(source, {
    renderDescription(node, ctx) {
      if (node.type === 'page') {
        const page = source.getNodePage(node, ctx.lang);
        if (page?.data.description) return cleanDescription(page.data.description);
      } else {
        const meta = source.getNodeMeta(node, ctx.lang);
        if (meta?.data.description) return cleanDescription(meta.data.description);
      }
      return typeof node.description === 'string' ? cleanDescription(node.description) : '';
    },
  }).index();
}
