
import "server-only";

const MAX_ATTEMPTS = 5;
const WINDOW_MS = 15 * 60 * 1000;
const BLOCK_DURATION_MS = 15 * 60 * 1000;

interface LoginAttempt {
  count: number;
  windowStartedAt: number;
  blockedUntil: number | null;
}

const attempts = new Map<string, LoginAttempt>();

export function getLoginIdentifier(request: Request): string {
  const forwardedFor = request.headers.get("x-forwarded-for");
  const realIp = request.headers.get("x-real-ip");

  const ip = forwardedFor?.split(",")[0]?.trim() || realIp || "local";

  return ip;
}

export function checkLoginRateLimit(identifier: string) {
  const now = Date.now();
  const attempt = attempts.get(identifier);

  if (!attempt) {
    return { allowed: true, retryAfter: 0 };
  }

  if (attempt.blockedUntil && now < attempt.blockedUntil) {
    return {
      allowed: false,
      retryAfter: Math.ceil((attempt.blockedUntil - now) / 1000),
    };
  }

  if (now - attempt.windowStartedAt >= WINDOW_MS) {
    attempts.delete(identifier);
  }

  return { allowed: true, retryAfter: 0 };
}

export function recordFailedLogin(identifier: string) {
  const now = Date.now();
  let attempt = attempts.get(identifier);

  if (!attempt || now - attempt.windowStartedAt >= WINDOW_MS) {
    attempt = {
      count: 0,
      windowStartedAt: now,
      blockedUntil: null,
    };
  }

  attempt.count += 1;

  if (attempt.count >= MAX_ATTEMPTS) {
    attempt.blockedUntil = now + BLOCK_DURATION_MS;
  }

  attempts.set(identifier, attempt);

  return {
    blocked: attempt.blockedUntil !== null,
    retryAfter: attempt.blockedUntil
      ? Math.ceil((attempt.blockedUntil - now) / 1000)
      : 0,
  };
}

export function resetLoginRateLimit(identifier: string) {
  attempts.delete(identifier);
}
