import { Gauge, House, Pulse, SignIn, SignOut, SquaresFour } from "@phosphor-icons/react";
import type { CSSProperties, ReactNode } from "react";
import logoUrl from "../assets/jetkzu-logo.png";
import type { ActivityItem, AppRoute, Session } from "../types";

type ProductShellProps = {
  route: AppRoute;
  title: string;
  description: string;
  session: Session;
  activity: ActivityItem[];
  onLogout: () => void;
  children: ReactNode;
};

const navItems = [
  { route: "landing" as const, href: "/", label: "Platform", icon: <House weight="bold" /> },
  { route: "dashboard" as const, href: "/dashboard", label: "Dashboard", icon: <SquaresFour weight="bold" /> }
];

export function ProductShell({ route, title, description, session, activity, onLogout, children }: ProductShellProps) {
  const isLoggedIn = Boolean(session.user);
  const visibleNavItems = isLoggedIn ? navItems : navItems.filter((item) => item.route !== "dashboard");

  return (
    <main className={route === "dashboard" ? "appShell" : "siteShell"}>
      <header className="siteHeader">
        <a className="brand" href="/" aria-label="JetKZu platform home">
          <span className="brandLogo">
            <img src={logoUrl} alt="JetKZu Taxi logo" />
          </span>
          <span>
            <strong>JetKZu</strong>
            <span>Taxi platform</span>
          </span>
        </a>

        <nav className="topNav" aria-label="Primary navigation">
          {visibleNavItems.map((item) => (
            <a key={item.route} className={route === item.route ? "active" : ""} href={item.href}>
              {item.icon}
              <span>{item.label}</span>
            </a>
          ))}
          {isLoggedIn ? (
            <button className="navButton" type="button" onClick={onLogout}>
              <SignOut weight="bold" />
              <span>Logout</span>
            </button>
          ) : (
            <a className={route === "login" ? "active" : ""} href="/login">
              <SignIn weight="bold" />
              <span>Login</span>
            </a>
          )}
        </nav>
      </header>

      <section className="pageFrame">
        <div className="pageIntro reveal">
          <div>
            <h1>{title}</h1>
            <p>{description}</p>
          </div>
          <div className="sessionBadge">
            <Gauge weight="bold" />
          <span>{session.user ? `${session.user.full_name} · ${session.app_role === "driver" ? "Таксист" : "Клиент"}` : "Гость"}</span>
          </div>
        </div>

        {children}

        {route === "dashboard" && <ActivityLog activity={activity} />}
      </section>
    </main>
  );
}

function ActivityLog({ activity }: { activity: ActivityItem[] }) {
  return (
    <section className="activityLog reveal" aria-label="Recent activity">
      <div className="sectionHeader">
        <strong>Лента действий</strong>
        <span>{activity.length ? "Последние ответы системы" : "Пока действий нет"}</span>
      </div>
      {activity.length === 0 ? (
        <div className="emptyState">
          <Pulse weight="bold" />
          <p>Создайте заказ, примите поездку, смените статус или проведите оплату, чтобы увидеть историю.</p>
        </div>
      ) : (
        <div className="activityList">
          {activity.map((item, index) => (
            <article key={item.id} className={item.result.ok ? "activityItem success" : "activityItem error"} style={{ "--index": index } as CSSProperties}>
              <div>
                <strong>{item.label}</strong>
                <span>{item.result.ok ? "Готово" : "Нужно внимание"}</span>
              </div>
              <p>{messageFromBody(item.result.body, item.result.ok)}</p>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function messageFromBody(body: unknown, ok: boolean) {
  if (body && typeof body === "object") {
    const data = body as Record<string, unknown>;
    const message = data.message ?? data.error ?? data.status;
    if (typeof message === "string" && message.trim()) return message;
  }
  return ok ? "JetKZu принял действие." : "Backend вернул ошибку для этого действия.";
}
