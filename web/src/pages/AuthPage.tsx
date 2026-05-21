import { Key, SignIn, UserPlus } from "@phosphor-icons/react";
import { useState } from "react";
import { api } from "../api";
import { ActionButton, Field, Surface } from "../components/ui";
import type { PageProps, Session } from "../types";

type FormErrors = Partial<Record<"email" | "password" | "fullName" | "phone", string>>;
type AuthPageProps = PageProps & {
  accessMessage?: string;
};

const validateEmail = (email: string) => /\S+@\S+\.\S+/.test(email);

export function AuthPage({ saveSession, record, accessMessage = "" }: AuthPageProps) {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("passenger@jetkzu.kz");
  const [password, setPassword] = useState("Password123");
  const [fullName, setFullName] = useState("Aigerim Sadykova");
  const [phone, setPhone] = useState("+7 701 184 4296");
  const [errors, setErrors] = useState<FormErrors>({});
  const [formError, setFormError] = useState("");
  const [loading, setLoading] = useState(false);

  const validate = () => {
    const next: FormErrors = {};
    if (!validateEmail(email)) next.email = "Use a valid email address.";
    if (password.length < 6) next.password = "Password must be at least 6 characters.";
    if (mode === "register" && fullName.trim().length < 3) next.fullName = "Enter the passenger full name.";
    if (mode === "register" && phone.trim().length < 8) next.phone = "Enter a reachable phone number.";
    setErrors(next);
    return Object.keys(next).length === 0;
  };

  const submit = async () => {
    setFormError("");
    if (!validate()) return;
    setLoading(true);
    try {
      if (mode === "login") {
        const result = await api("/api/auth/login", { method: "POST", body: JSON.stringify({ email, password }) });
        record("Signed in", result);
        if (!result.ok) {
          setFormError("Login failed. Check the credentials or register first.");
          return;
        }
        saveSession(result.body as Session);
        window.location.href = "/dashboard";
        return;
      }

      const result = await api("/api/auth/register", {
        method: "POST",
        body: JSON.stringify({ email, password, full_name: fullName, phone, role: "passenger" })
      });
      record("Created passenger account", result);
      if (!result.ok) {
        setFormError("Registration failed. Try another email or check the backend response.");
        return;
      }
      setMode("login");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="authPage">
      <Surface title={mode === "login" ? "Welcome back" : "Create passenger account"} eyebrow="Account" icon={mode === "login" ? <SignIn weight="bold" /> : <UserPlus weight="bold" />}>
        {accessMessage && <p className="formNotice">{accessMessage}</p>}
        <div className="segmented" aria-label="Authentication mode">
          <button className={mode === "login" ? "active" : "secondaryButton"} type="button" onClick={() => setMode("login")}>Login</button>
          <button className={mode === "register" ? "active" : "secondaryButton"} type="button" onClick={() => setMode("register")}>Register</button>
        </div>

        <form className="formGrid" noValidate onSubmit={(event) => { event.preventDefault(); void submit(); }}>
          <Field label="Email" value={email} onChange={setEmail} type="email" error={errors.email} helper="Use the email you registered with." />
          <Field label="Password" value={password} onChange={setPassword} type="password" error={errors.password} helper="Minimum 6 characters." />
          {mode === "register" && (
            <>
              <Field label="Full name" value={fullName} onChange={setFullName} error={errors.fullName} helper="Shown in ride and payment records." />
              <Field label="Phone" value={phone} onChange={setPhone} type="tel" error={errors.phone} helper="Used for passenger contact details." />
            </>
          )}

          {formError && <p className="formError">{formError}</p>}

          <div className="actions">
            <ActionButton icon={<Key weight="bold" />} label={mode === "login" ? "Login" : "Register"} buttonType="submit" loading={loading} />
          </div>
        </form>
      </Surface>

      <aside className="authAside reveal">
        <strong>Your rides stay private until you sign in.</strong>
        <p>Dashboard access is protected. After login, JetKZu keeps your route, payment, receipt, and ride messages together.</p>
      </aside>
    </div>
  );
}
