export type ApiResult = { ok: boolean; status: number; body: unknown };

export const authHeaders = (token = ""): Record<string, string> =>
  token ? { Authorization: `Bearer ${token}` } : {};

export const api = async (path: string, options: RequestInit = {}, token = ""): Promise<ApiResult> => {
  const res = await fetch(path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(token),
      ...(options.headers ?? {})
    }
  });
  const text = await res.text();
  const body = text ? JSON.parse(text) : null;
  return { ok: res.ok, status: res.status, body };
};
