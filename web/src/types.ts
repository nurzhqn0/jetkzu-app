import type { ApiResult } from "./api";

export type User = {
  id: string;
  email: string;
  full_name: string;
  phone?: string;
  role: string;
};

export type AppRole = "passenger" | "driver";

export type Session = {
  access_token?: string;
  user?: User;
  app_role?: AppRole;
};

export type ActivityItem = {
  id: string;
  label: string;
  result: ApiResult;
};

export type AppRoute = "landing" | "login" | "dashboard";

export type PageProps = {
  token: string;
  session: Session;
  saveSession: (session: Session) => void;
  record: (label: string, result: ApiResult) => void;
};
