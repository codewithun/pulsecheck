import { NextRequest, NextResponse } from "next/server";

import { verifySession } from "@/lib/auth/session";
import { getTargets } from "@/lib/api/targets";
import { CentralApiError } from "@/lib/api/central-api";

const VALID_STATUSES = ["UP", "DOWN", "UNKNOWN"];
const VALID_PERIODS = ["1h", "24h", "7d", "30d"];

export async function GET(request: NextRequest) {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json(
      { error: "Unauthorized" },
      { status: 401 },
    );
  }

  try {
    const params = request.nextUrl.searchParams;

    const limit = Number(params.get("limit") ?? 10);
    const offset = Number(params.get("offset") ?? 0);
    const status = params.get("status") ?? undefined;
    const probe_id = params.get("probe_id") ?? undefined;
    const search = params.get("search") ?? undefined;
    const period = params.get("period") ?? undefined;

    if (
      !Number.isInteger(limit) ||
      limit < 1 ||
      limit > 100 ||
      !Number.isInteger(offset) ||
      offset < 0
    ) {
      return NextResponse.json(
        { error: "Invalid pagination parameters" },
        { status: 400 },
      );
    }

    if (status && !VALID_STATUSES.includes(status)) {
      return NextResponse.json(
        { error: "Invalid status parameter" },
        { status: 400 },
      );
    }

    if (period && !VALID_PERIODS.includes(period)) {
      return NextResponse.json(
        { error: "Invalid period parameter" },
        { status: 400 },
      );
    }

    const result = await getTargets({
      limit,
      offset,
      status: status as "UP" | "DOWN" | "UNKNOWN" | undefined,
      probe_id,
      search,
      period: period as "1h" | "24h" | "7d" | "30d" | undefined,
    });

    return NextResponse.json(result, {
      headers: {
        "Cache-Control": "no-store",
      },
    });
  } catch (error) {
    if (error instanceof CentralApiError) {
      return NextResponse.json(
        { error: "Central API request failed" },
        { status: 502 },
      );
    }

    console.error("Targets API route error:", error);

    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}
