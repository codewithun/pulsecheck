
import { Suspense, type ReactNode } from "react";
import { connection } from "next/server";
import { requireSession } from "@/lib/auth/session";

async function AuthGuard({ children }: { children: ReactNode }) {
  await connection();
  await requireSession();

  return <>{children}</>;
}

export default function ProtectedLayout({
  children,
}: {
  children: ReactNode;
}) {
  return (
    <Suspense
      fallback={
        <main className="flex min-h-screen items-center justify-center bg-slate-50">
          <p className="text-sm text-slate-500">Verifying session...</p>
        </main>
      }
    >
      <AuthGuard>{children}</AuthGuard>
    </Suspense>
  );
}
