import { NextRequest, NextResponse } from "next/server";

import { verifySession } from "@/lib/auth/session";
import {
  createTargetGroup,
  getTargetGroups,
} from "@/lib/api/target-metadata";
import { CentralApiError } from "@/lib/api/central-api";

export async function GET() {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const result = await getTargetGroups();

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

    console.error("Target groups GET error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}

export async function POST(request: NextRequest) {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
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

    const result = await createTargetGroup({
      name: body.name.trim(),
      description: body.description.trim(),
    });

    return NextResponse.json(result, { status: 201 });
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Target groups POST error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}