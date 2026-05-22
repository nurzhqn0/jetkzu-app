import { Key, SteeringWheel, UserCircle } from "@phosphor-icons/react";
import { useState } from "react";
import { api } from "../api";
import { ActionButton, Field, Surface } from "../components/ui";
import type { AppRole, PageProps, Session } from "../types";

type FormErrors = Partial<Record<"fullName" | "phone", string>>;
type AuthPageProps = PageProps & {
  accessMessage?: string;
};

const normalizePhone = (phone: string) => phone.replace(/\D/g, "");

const credentialsFromPhone = (phone: string) => {
  const digits = normalizePhone(phone);
  const safeDigits = digits || "70000000000";
  return {
    email: `${safeDigits}@phone.jetkzu.local`,
    password: `JetKZu-${safeDigits.slice(-6).padStart(6, "0")}`
  };
};

const roleLabel = (role: AppRole) => (role === "driver" ? "таксист" : "клиент");

export function AuthPage({ saveSession, record, accessMessage = "" }: AuthPageProps) {
  const [role, setRole] = useState<AppRole>("passenger");
  const [fullName, setFullName] = useState("Dias Aitugan");
  const [phone, setPhone] = useState("+7 701 000 2026");
  const [errors, setErrors] = useState<FormErrors>({});
  const [formError, setFormError] = useState("");
  const [loading, setLoading] = useState(false);

  const validate = () => {
    const next: FormErrors = {};
    if (fullName.trim().length < 2) next.fullName = "Введите имя.";
    if (normalizePhone(phone).length < 10) next.phone = "Введите номер телефона.";
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const loginWithGeneratedCredentials = async (): Promise<Session | null> => {
    const { email, password } = credentialsFromPhone(phone);
    const login = await api("/api/auth/login", { method: "POST", body: JSON.stringify({ email, password }) });
    record("Вход по телефону", login);
    if (login.ok) return login.body as Session;

    const register = await api("/api/auth/register", {
      method: "POST",
      body: JSON.stringify({
        email,
        password,
        full_name: fullName.trim(),
        phone: phone.trim(),
        role
      })
    });
    record(`Регистрация: ${roleLabel(role)}`, register);
    if (!register.ok) return null;

    const retry = await api("/api/auth/login", { method: "POST", body: JSON.stringify({ email, password }) });
    record("Автовход после регистрации", retry);
    return retry.ok ? (retry.body as Session) : null;
  };

  const submit = async () => {
    setFormError("");
    if (!validate()) return;
    setLoading(true);
    try {
      const nextSession = await loginWithGeneratedCredentials();
      if (!nextSession?.access_token || !nextSession.user) {
        setFormError("Не получилось войти. Проверьте, что backend запущен, или попробуйте другой номер.");
        return;
      }

      const token = nextSession.access_token;
      const userId = nextSession.user.id;

      await api("/api/users/me", {
        method: "PUT",
        body: JSON.stringify({ full_name: fullName.trim(), phone: phone.trim() })
      }, token);

      if (nextSession.user.role !== role) {
        const roleUpdate = await api(`/api/users/${userId}/role`, {
          method: "PATCH",
          body: JSON.stringify({ role })
        }, token);
        record(`Роль выбрана: ${roleLabel(role)}`, roleUpdate);
      }

      saveSession({
        ...nextSession,
        app_role: role,
        user: {
          ...nextSession.user,
          full_name: fullName.trim(),
          phone: phone.trim(),
          role
        }
      });
      window.location.href = "/dashboard";
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="authPage">
      <Surface title="Кто сегодня едет?" eyebrow="Вход" icon={role === "driver" ? <SteeringWheel weight="bold" /> : <UserCircle weight="bold" />}>
        {accessMessage && <p className="formNotice">{accessMessage}</p>}
        <div className="segmented roleSwitch" aria-label="Выбор роли">
          <button className={role === "passenger" ? "active" : "secondaryButton"} type="button" onClick={() => setRole("passenger")}>
            Клиент
          </button>
          <button className={role === "driver" ? "active" : "secondaryButton"} type="button" onClick={() => setRole("driver")}>
            Таксист
          </button>
        </div>

        <form className="formGrid" noValidate onSubmit={(event) => { event.preventDefault(); void submit(); }}>
          <Field label="Имя" value={fullName} onChange={setFullName} error={errors.fullName} placeholder="Например, Dias" />
          <Field label="Телефон" value={phone} onChange={setPhone} type="tel" error={errors.phone} placeholder="+7 701 000 2026" />

          {formError && <p className="formError">{formError}</p>}

          <div className="authRolePreview">
            {role === "driver" ? <SteeringWheel weight="bold" /> : <UserCircle weight="bold" />}
            <span>{role === "driver" ? "Откроется список заказов и управление поездкой." : "Откроется заказ такси с выбором маршрута и оплаты."}</span>
          </div>

          <div className="actions">
            <ActionButton icon={<Key weight="bold" />} label={loading ? "Входим" : "Продолжить"} buttonType="submit" loading={loading} />
          </div>
        </form>
      </Surface>

      <aside className="authAside reveal">
        <strong>JetKZu Taxi</strong>
        <p>Один вход открывает нужный экран: клиент заказывает поездку, таксист принимает заказ и ведёт маршрут до завершения.</p>
      </aside>
    </div>
  );
}
