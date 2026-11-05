
import "server-only";

import { compare } from "bcryptjs";
import { NextRequest, NextResponse } from "next/server";
import { createSession } from "@/lib/auth/session";

export async function POST(request: NextRequest) {
  try {
    // Basic same-origin check to protect the login endpoint.
    const origin = request.headers.get("origin");
    const host = request.headers.get("host");

    if (origin && host) {
      const originHost = new URL(origin).host;

      if (originHost !== host) {
        return NextResponse.json(
          { error: "Forbidden" },
          { status: 403 },
        );
      }
    }

    const body: unknown = await request.json();

    if (
      typeof body !== "object" ||
      body === null ||
      !("username" in body) ||
      !("password" in body) ||
      typeof body.username !== "string" ||
      typeof body.password !== "string"
    ) {
      return NextResponse.json(
        { error: "Invalid request" },
        { status: 400 },
      );
    }

    const { username, password } = body;

    const adminUsername = process.env.DASHBOARD_USERNAME;
    const passwordHash = process.env.DASHBOARD_PASSWORD_HASH;

    if (!adminUsername || !passwordHash) {
      console.error("Dashboard credentials are not configured");

      return NextResponse.json(
        { error: "Authentication is unavailable" },
        { status: 500 },
      );
    }

    const usernameValid = username === adminUsername;
    const passwordValid = await compare(password, passwordHash);

    if (!usernameValid || !passwordValid) {
      return NextResponse.json(
        { error: "Invalid username or password" },
        { status: 401 },
      );
    }

    await createSession(adminUsername);

    return NextResponse.json({
      message: "Login successful",
    });
  } catch (error) {
    console.error("Login error:", error);

    return NextResponse.json(
      { error: "Internal server error" },
      { status: 500 },
    );
  }
}
