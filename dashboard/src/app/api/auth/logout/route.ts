
import "server-only";

import { NextRequest, NextResponse } from "next/server";
import { deleteSession } from "@/lib/auth/session";

export async function POST(request: NextRequest) {
  const origin = request.headers.get("origin");
  const host = request.headers.get("host");

  if (!origin || !host) {
    return NextResponse.json(
      { error: "Forbidden" },
      { status: 403 },
    );
  }

  try {
    if (new URL(origin).host !== host) {
      return NextResponse.json(
        { error: "Forbidden" },
        { status: 403 },
      );
    }
  } catch {
    return NextResponse.json(
      { error: "Invalid origin" },
      { status: 403 },
    );
  }

  await deleteSession();

  return NextResponse.json({
    message: "Logout successful",
  });
}
