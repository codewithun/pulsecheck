import { NextRequest, NextResponse } from "next/server";
import { verifySession } from "@/lib/auth/session";
import { centralApiFetch, CentralApiError } from "@/lib/api/central-api";
import type { OverviewPeriod, OverviewResponse } from "@/lib/api/types";

const ALLOWED_PERIODS: OverviewPeriod[] = ["1h", "24h", "7d", "30d"];

export async function GET(request: NextRequest) {
  const session = await verifySession();

  if (!session) {
    return NextResponse.json(
      { error: "unauthorized" },
      { status: 401 },
    );
  }

  const requestedPeriod = request.nextUrl.searchParams.get("period") ?? "24h";

  if (!ALLOWED_PERIODS.includes(requestedPeriod as OverviewPeriod)) {
    return NextResponse.json(
      { error: "invalid period" },
      { status: 400 },
    );
  }

  try {
    const data = await centralApiFetch<OverviewResponse>(
      `/api/v1/overview?period=${requestedPeriod}`,
    );

    return NextResponse.json(data);
  } catch (error) {
    if (error instanceof CentralApiError) {
      console.error("Central API error:", error.status);

      return NextResponse.json(
        { error: "failed to retrieve overview" },
        { status: 502 },
      );
    }

    console.error("Overview proxy error:", error);

    return NextResponse.json(
      { error: "internal server error" },
      { status: 500 },
    );
  }
}
