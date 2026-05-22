import { useEffect, useMemo, useState } from "react";
import type { ApiResult } from "./api";
import { ProductShell } from "./components/ProductShell";
import { AuthPage } from "./pages/AuthPage";
import { DashboardPage } from "./pages/DashboardPage";
import { LandingPage } from "./pages/LandingPage";
import type { ActivityItem, AppRoute, Session } from "./types";

const routeFromPath = (path: string): AppRoute => {
  if (path === "/login") return "login";
  if (path === "/dashboard") return "dashboard";
  return "landing";
};

const routeMeta: Record<AppRoute, { title: string; description: string }> = {
  landing: {
    title: "JetKZu такси в браузере",
    description: "Клиент заказывает поездку, таксист принимает заказ, статусы и оплата проходят в одном понятном сценарии."
  },
  login: {
    title: "Вход по телефону",
    description: "Укажите имя, номер и роль: клиент или таксист. Остальные технические данные JetKZu заполнит сам."
  },
  dashboard: {
    title: "Рабочий экран JetKZu",
    description: "Без ручного ввода ID: только маршрут, заказ, принятие поездки, статусы и демо-оплата."
  }
};

export function App() {
  const [route, setRoute] = useState<AppRoute>(() => routeFromPath(window.location.pathname));
  const [session, setSession] = useState<Session>(() => {
    const raw = localStorage.getItem("jetkzu-session");
    return raw ? JSON.parse(raw) : {};
  });
  const [activity, setActivity] = useState<ActivityItem[]>([]);
  const [accessMessage, setAccessMessage] = useState("");

  useEffect(() => {
    const syncRoute = () => setRoute(routeFromPath(window.location.pathname));
    window.addEventListener("popstate", syncRoute);
    return () => window.removeEventListener("popstate", syncRoute);
  }, []);

  const token = session.access_token ?? "";
  const isLoggedIn = Boolean(token && session.user);
  const visibleRoute: AppRoute = route === "dashboard" && !isLoggedIn ? "login" : route;
  const meta = routeMeta[visibleRoute];

  useEffect(() => {
    if (route === "dashboard" && !isLoggedIn) {
      setAccessMessage("Login first to open the dashboard.");
      window.history.replaceState({}, "", "/login");
      setRoute("login");
    }
  }, [isLoggedIn, route]);

  const saveSession = (next: Session) => {
    setAccessMessage("");
    setSession(next);
    localStorage.setItem("jetkzu-session", JSON.stringify(next));
  };

  const logout = () => {
    setSession({});
    localStorage.removeItem("jetkzu-session");
    window.history.pushState({}, "", "/");
    setRoute("landing");
  };

  const record = (label: string, result: ApiResult) => {
    setActivity((items) => [{ id: crypto.randomUUID(), label, result }, ...items].slice(0, 8));
  };

  const pageProps = useMemo(() => ({ token, session, saveSession, record }), [token, session]);

  return (
    <ProductShell route={visibleRoute} title={meta.title} description={meta.description} session={session} activity={activity} onLogout={logout}>
      {visibleRoute === "landing" && <LandingPage session={session} />}
      {visibleRoute === "login" && <AuthPage {...pageProps} accessMessage={accessMessage} />}
      {visibleRoute === "dashboard" && <DashboardPage {...pageProps} />}
    </ProductShell>
  );
}
