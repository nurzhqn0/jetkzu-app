import type { ApiResult } from "./api";

export type AppRoute = "landing" | "login" | "dashboard";

export type User = {
  id: string;
  email: string;
  full_name?: string;
  phone?: string;
  role?: string;
};

export type Session = {
  user?: User;
  access_token?: string;
  refresh_token?: string;
  expires_at?: string;
};

export type ActivityItem = {
  id: string;
  label: string;
  result: ApiResult;
};

export type PageProps = {
  token: string;
  session: Session;
  saveSession: (session: Session) => void;
  record: (label: string, result: ApiResult) => void;
};
