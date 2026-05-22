import { describe, expect, it } from "vitest";
import { authHeaders } from "./api";

describe("authHeaders", () => {
  it("creates a bearer token header when a token is present", () => {
    expect(authHeaders("abc123")).toEqual({ Authorization: "Bearer abc123" });
  });

  it("returns an empty object without a token", () => {
    expect(authHeaders()).toEqual({});
  });
});
