import type { BaseLayoutProps } from 'fumadocs-ui/layouts/shared';
import { BookText } from 'lucide-react';
import Image from 'next/image';
import { appName, asset, pkgGoDev, repoUrl } from './shared';

export function baseOptions(): BaseLayoutProps {
  return {
    nav: {
      title: (
        <>
          <Image src={asset('/logo.png')} alt="" width={24} height={24} className="size-6 rounded-sm" priority />
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
