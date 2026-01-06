import Link from "next/link";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-24">
      <div className="max-w-2xl text-center">
        <h1 className="text-4xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-6xl">
          Uptime Kuma
        </h1>
        <p className="mt-6 text-lg leading-8 text-gray-600 dark:text-gray-300">
          Self-hosted monitoring tool - Rewritten in Go + Next.js
        </p>
        <div className="mt-10 flex items-center justify-center gap-x-6">
          <Link
            href="/setup"
            className="rounded-md bg-primary px-3.5 py-2.5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
          >
            Get Started
          </Link>
          <Link
            href="/login"
            className="text-sm font-semibold leading-6 text-gray-900 dark:text-white"
          >
            Login <span aria-hidden="true">→</span>
          </Link>
        </div>
      </div>
    </div>
  );
}
