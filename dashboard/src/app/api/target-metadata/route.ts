import { NextRequest, NextResponse } from "next/server";

import { verifySession } from "@/lib/auth/session";
import {
  getTargetMetadata,
  updateTargetMetadata,
} from "@/lib/api/target-metadata";
import { CentralApiError } from "@/lib/api/central-api";

export async function GET(request: NextRequest) {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const params = request.nextUrl.searchParams;
    const probeId = params.get("probe_id");
    const target = params.get("target");

    if (!probeId?.trim() || !target?.trim()) {
      return NextResponse.json(
        { error: "probe_id and target are required" },
        { status: 400 },
      );
    }

    const result = await getTargetMetadata(probeId, target);

    return NextResponse.json(result, {
      headers: { "Cache-Control": "no-store" },
    });
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Target metadata GET error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}

export async function PUT(request: NextRequest) {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const body = await request.json();

    if (
      typeof body.probe_id !== "string" ||
      !body.probe_id.trim() ||
      body.probe_id.length > 100 ||
      typeof body.target !== "string" ||
      !body.target.trim() ||
      !(
        body.group_id === null ||
        (Number.isSafeInteger(body.group_id) && body.group_id > 0)
      ) ||
      !Array.isArray(body.tags) ||
      body.tags.length > 50 ||
      !body.tags.every(
        (id: unknown) =>
          Number.isSafeInteger(id) && Number(id) > 0,
      ) ||
      new Set(body.tags).size !== body.tags.length
    ) {
      return NextResponse.json(
        { error: "Invalid target metadata" },
        { status: 400 },
      );
    }

    const result = await updateTargetMetadata({
      probe_id: body.probe_id.trim(),
      target: body.target.trim(),
      group_id: body.group_id,
      tags: body.tags,
    });

    return NextResponse.json(result);
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Target metadata PUT error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}