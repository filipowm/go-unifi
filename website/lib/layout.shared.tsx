import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { BookText, Code2 } from 'lucide-react';
import { appName, pkgGoDev, repoUrl } from './shared';

export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: (
        <>
          <Code2 className="size-5" />
          <span className="font-semibold">{appName}</span>
        </>
      ),
    },
    links: [
      {
        text: 'Documentation',
        url: '/docs',
        active: 'nested-url',
      },
      {
        icon: <BookText />,
        text: 'Go API (pkg.go.dev)',
        url: pkgGoDev,
        external: true,
      },
    ],
    githubUrl: repoUrl,
  };
}
