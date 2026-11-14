import { NextRequest, NextResponse } from "next/server";

import { verifySession } from "@/lib/auth/session";
import {
  createTargetTag,
  getTargetTags,
} from "@/lib/api/target-metadata";
import { CentralApiError } from "@/lib/api/central-api";

export async function GET() {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json({ error: "Unauthorized" }, { status: 401 });
  }

  try {
    const result = await getTargetTags();

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

    console.error("Target tags GET error:", error);
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
      !body.name.trim() ||
      body.name.trim().length > 50
    ) {
      return NextResponse.json(
        { error: "Invalid tag data" },
        { status: 400 },
      );
    }

    const result = await createTargetTag({
      name: body.name.trim(),
    });

    return NextResponse.json(result, { status: 201 });
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Target tags POST error:", error);
    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}