import Image from 'next/image';
import Link from 'next/link';
import { repoUrl, pkgGoDev } from '@/lib/shared';

export default function HomePage() {
  return (
    <main className="relative flex flex-1 flex-col items-center justify-center overflow-hidden px-4 py-20 text-center">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(60%_50%_at_50%_0%,rgba(0,173,216,0.16),transparent_70%)]"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(45%_45%_at_75%_15%,rgba(5,89,201,0.12),transparent_70%)]"
      />

      <Image
        src="/logo.png"
        alt="go-unifi logo"
        width={150}
        height={150}
        priority
        className="mb-6 drop-shadow-[0_8px_30px_rgba(0,173,216,0.35)]"
      />
      <span className="mb-4 rounded-full border px-3 py-1 text-xs font-medium text-fd-muted-foreground">
        Go client for the UniFi Network controller API
      </span>
      <h1 className="max-w-3xl bg-gradient-to-r from-[#00add8] to-[#0559c9] bg-clip-text pb-2 text-4xl font-bold tracking-tight text-transparent sm:text-5xl">
        go-unifi
      </h1>
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
          href="/docs/reference"
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
