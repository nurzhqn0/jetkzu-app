import type { ReactNode } from "react";

type FieldProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  placeholder?: string;
  helper?: string;
  error?: string;
};

export function Field({ label, value, onChange, type = "text", placeholder, helper, error }: FieldProps) {
  return (
    <label className="field">
      <span>{label}</span>
      <input
        aria-invalid={Boolean(error)}
        type={type}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
      {helper && !error && <small>{helper}</small>}
      {error && <small className="fieldError">{error}</small>}
    </label>
  );
}

export function ActionButton({
  icon,
  label,
  onClick,
  variant = "primary",
  disabled = false,
  loading = false,
  buttonType = "button"
}: {
  icon: ReactNode;
  label: string;
  onClick?: () => void;
  variant?: "primary" | "secondary";
  disabled?: boolean;
  loading?: boolean;
  buttonType?: "button" | "submit";
}) {
  return (
    <button className={variant === "secondary" ? "secondaryButton" : ""} type={buttonType} onClick={onClick} disabled={disabled || loading} title={label}>
      {loading ? <span className="buttonLoader" aria-hidden="true" /> : icon}
      <span>{loading ? "Working" : label}</span>
    </button>
  );
}

export function Surface({ title, eyebrow, icon, children, wide = false }: { title: string; eyebrow?: string; icon?: ReactNode; children: ReactNode; wide?: boolean }) {
  return (
    <section className={wide ? "surface wide" : "surface"}>
      <div className="surfaceHeader">
        {icon}
        <div>
          {eyebrow && <span>{eyebrow}</span>}
          <h2>{title}</h2>
        </div>
      </div>
      {children}
    </section>
  );
}

export function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

export function StatusBadge({ children, tone = "neutral" }: { children: ReactNode; tone?: "neutral" | "success" | "warning" | "blue" }) {
  return <span className={`statusBadge ${tone}`}>{children}</span>;
}
