
import "server-only";

import { compare } from "bcryptjs";
import { NextRequest, NextResponse } from "next/server";

import { createSession } from "@/lib/auth/session";
import {
  checkLoginRateLimit,
  getLoginIdentifier,
  recordFailedLogin,
  resetLoginRateLimit,
} from "@/lib/auth/rate-limit";

function jsonResponse(
  body: Record<string, string>,
  status = 200,
  retryAfter?: number,
) {
  const headers = new Headers({
    "Cache-Control": "no-store",
  });

  if (retryAfter) {
    headers.set("Retry-After", String(retryAfter));
  }

  return NextResponse.json(body, { status, headers });
}

export async function POST(request: NextRequest) {
  try {
    // Require a valid same-origin request.
    const origin = request.headers.get("origin");

    if (!origin || new URL(origin).origin !== request.nextUrl.origin) {
      return jsonResponse({ error: "Forbidden" }, 403);
    }

    const fetchSite = request.headers.get("sec-fetch-site");

    if (fetchSite && fetchSite !== "same-origin") {
      return jsonResponse({ error: "Forbidden" }, 403);
    }

    const identifier = getLoginIdentifier(request);
    const rateLimit = checkLoginRateLimit(identifier);

    if (!rateLimit.allowed) {
      return jsonResponse(
        { error: "Too many login attempts. Try again later." },
        429,
        rateLimit.retryAfter,
      );
    }

    let body: unknown;

    try {
      body = await request.json();
    } catch {
      return jsonResponse({ error: "Invalid request" }, 400);
    }

    if (
      typeof body !== "object" ||
      body === null ||
      !("username" in body) ||
      !("password" in body) ||
      typeof body.username !== "string" ||
      typeof body.password !== "string"
    ) {
      return jsonResponse({ error: "Invalid request" }, 400);
    }

    const { username, password } = body;

    if (
      username.length < 1 ||
      username.length > 128 ||
      password.length < 1 ||
      password.length > 128
    ) {
      return jsonResponse({ error: "Invalid request" }, 400);
    }

    const adminUsername = process.env.DASHBOARD_USERNAME;
    const passwordHash = process.env.DASHBOARD_PASSWORD_HASH;

    if (!adminUsername || !passwordHash) {
      console.error("Dashboard credentials are not configured");

      return jsonResponse(
        { error: "Authentication is unavailable" },
        500,
      );
    }

    const usernameValid = username === adminUsername;
    const passwordValid = await compare(password, passwordHash);

    if (!usernameValid || !passwordValid) {
      const result = recordFailedLogin(identifier);

      if (result.blocked) {
        return jsonResponse(
          { error: "Too many login attempts. Try again later." },
          429,
          result.retryAfter,
        );
      }

      return jsonResponse(
        { error: "Invalid username or password" },
        401,
      );
    }

    resetLoginRateLimit(identifier);

    await createSession(adminUsername);

    return jsonResponse({ message: "Login successful" });
  } catch (error) {
    console.error("Login error:", error);

    return jsonResponse(
      { error: "Internal server error" },
      500,
    );
  }
}
