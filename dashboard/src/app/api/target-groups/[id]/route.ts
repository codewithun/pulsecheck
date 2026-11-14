import { NextResponse } from "next/server";

import { verifySession } from "@/lib/auth/session";
import {
  deleteTargetGroup,
  updateTargetGroup,
} from "@/lib/api/target-metadata";
import { CentralApiError } from "@/lib/api/central-api";

type RouteContext = {
  params: Promise<{ id: string }>;
};

export async function PATCH(
  request: Request,
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
      return NextResponse.json({ error: "Invalid group ID" }, { status: 400 });
    }

    const body = await request.json();

    if (
      typeof body.name !== "string" ||
      typeof body.description !== "string" ||
      !body.name.trim() ||
      body.name.trim().length > 100
    ) {
      return NextResponse.json(
        { error: "Invalid group data" },
        { status: 400 },
      );
    }

    const result = await updateTargetGroup(id, {
      name: body.name.trim(),
      description: body.description.trim(),
    });

    return NextResponse.json(result);
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Target group PATCH error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}

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
      return NextResponse.json({ error: "Invalid group ID" }, { status: 400 });
    }

    await deleteTargetGroup(id);

    return new NextResponse(null, { status: 204 });
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Target group DELETE error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}