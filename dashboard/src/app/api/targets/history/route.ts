
import { NextRequest, NextResponse } from "next/server";

import { verifySession } from "@/lib/auth/session";
import { getTargetHistory } from "@/lib/api/target-history";
import { CentralApiError } from "@/lib/api/central-api";

const VALID_PERIODS = ["1h", "24h", "7d", "30d"];

export async function GET(request: NextRequest) {
  try {
    const session = await verifySession();

    if (!session) {
      return NextResponse.json(
        { error: "Unauthorized" },
        { status: 401 },
      );
    }

    const params = request.nextUrl.searchParams;

    const probe_id = params.get("probe_id")?.trim();
    const target = params.get("target");
    const period = params.get("period") ?? undefined;
    const limit = Number(params.get("limit") ?? 20);
    const offset = Number(params.get("offset") ?? 0);

    if (!probe_id || !target) {
      return NextResponse.json(
        { error: "probe_id and target are required" },
        { status: 400 },
      );
    }

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

    if (period && !VALID_PERIODS.includes(period)) {
      return NextResponse.json(
        { error: "Invalid period parameter" },
        { status: 400 },
      );
    }

    const result = await getTargetHistory({
      probe_id,
      target,
      limit,
      offset,
      period: period as "1h" | "24h" | "7d" | "30d" | undefined,
    });

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

    console.error("Target history API route error:", error);

    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}
