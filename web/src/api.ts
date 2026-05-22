export type ApiResult = {
  ok: boolean;
  status: number;
  body: unknown;
};

export const authHeaders = (token = ""): Record<string, string> =>
  token ? { Authorization: `Bearer ${token}` } : {};

export async function api(path: string, init: RequestInit = {}, token = ""): Promise<ApiResult> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  for (const [key, value] of Object.entries(authHeaders(token))) {
    headers.set(key, value);
  }

  const response = await fetch(path, { ...init, headers });
  const text = await response.text();
  let body: unknown = text;

  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = text;
    }
  }

  return {
    ok: response.ok,
    status: response.status,
    body
  };
}
