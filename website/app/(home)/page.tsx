import Link from 'next/link';
import { repoUrl, pkgGoDev } from '@/lib/shared';

export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col items-center justify-center px-4 py-20 text-center">
      <span className="mb-4 rounded-full border px-3 py-1 text-xs font-medium text-fd-muted-foreground">
        Go client for the UniFi Network controller API
      </span>
      <h1 className="max-w-3xl text-4xl font-bold tracking-tight sm:text-5xl">go-unifi</h1>
      <p className="mt-4 max-w-2xl text-fd-muted-foreground sm:text-lg">
        Type-safe access to networks, clients, devices, firewall rules and dozens of other UniFi resources —
        across both the Internal and the Official UniFi APIs, from one client.
      </p>

      <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
        <Link
          href="/docs"
          className="rounded-lg bg-fd-primary px-5 py-2.5 text-sm font-medium text-fd-primary-foreground transition-opacity hover:opacity-90"
        >
          Get started
        </Link>
        <Link
          href="/docs/reference/api"
          className="rounded-lg border px-5 py-2.5 text-sm font-medium transition-colors hover:bg-fd-accent"
        >
          API reference
        </Link>
        <a
          href={repoUrl}
          className="rounded-lg border px-5 py-2.5 text-sm font-medium transition-colors hover:bg-fd-accent"
        >
          GitHub
        </a>
      </div>

      <pre className="mt-12 overflow-x-auto rounded-lg border bg-fd-card px-5 py-4 text-left text-sm">
        <code>{`go get github.com/filipowm/go-unifi/v2`}</code>
      </pre>

      <p className="mt-6 text-sm text-fd-muted-foreground">
        Prefer GoDoc?{' '}
        <a href={pkgGoDev} className="font-medium underline underline-offset-4">
          Browse the API on pkg.go.dev
        </a>
        .
      </p>
    </main>
  );
}
