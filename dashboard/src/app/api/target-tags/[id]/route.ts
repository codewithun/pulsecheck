import { NextResponse } from "next/server";

import { verifySession } from "@/lib/auth/session";
import { deleteTargetTag } from "@/lib/api/target-metadata";
import { CentralApiError } from "@/lib/api/central-api";

type RouteContext = {
  params: Promise<{ id: string }>;
};

export async function DELETE(
  _request: Request,
  context: RouteContext,
) {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const { id: rawId } = await context.params;
    const id = Number(rawId);

    if (!Number.isSafeInteger(id) || id <= 0) {
      return NextResponse.json({ error: "Invalid tag ID" }, { status: 400 });
    }

    await deleteTargetTag(id);

    return new NextResponse(null, { status: 204 });
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Target tag DELETE error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}