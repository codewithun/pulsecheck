import "server-only";

const CENTRAL_API_URL = process.env.CENTRAL_API_URL;
const API_TOKEN = process.env.API_TOKEN;

export class CentralApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
  ) {
    super(message);
    this.name = "CentralApiError";
  }
}

export async function centralApiFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  if (!CENTRAL_API_URL || !API_TOKEN) {
    throw new Error("Central API environment variables are not configured");
  }

  if (!path.startsWith("/api/v1/")) {
    throw new Error("Invalid Central API path");
  }

  const baseUrl = CENTRAL_API_URL.replace(/\/+$/, "");
  const url = `${baseUrl}${path}`;

  const response = await fetch(url, {
    ...init,
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${API_TOKEN}`,
      ...init.headers,
    },
    cache: "no-store",
  });

  if (!response.ok) {
    throw new CentralApiError(
      `Central API request failed with status ${response.status}`,
      response.status,
    );
  }

  return response.json() as Promise<T>;
}
