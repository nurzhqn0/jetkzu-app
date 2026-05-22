import { ArrowRight, CarProfile, CreditCard, ShieldCheck } from "@phosphor-icons/react";
import type { Session } from "../types";

export function LandingPage({ session }: { session: Session }) {
  const href = session.user ? "/dashboard" : "/login";
  const label = session.user ? "Open dashboard" : "Start riding";

  return (
    <div className="landingLayout">
      <section className="hero reveal">
        <div>
          <span className="statusBadge warning">Kazakhstan taxi ops</span>
          <h2>One dashboard for rides, drivers, payments, and receipts.</h2>
          <p>
            JetKZu connects passenger booking, driver availability, trip status, payment capture,
            and notification history through a single local microservice stack.
          </p>
          <div className="heroActions">
            <a className="buttonLink" href={href}>
              <ArrowRight weight="bold" />
              <span>{label}</span>
            </a>
            <a className="buttonLink secondaryButton" href="/dashboard">
              <CarProfile weight="bold" />
              <span>View trip tools</span>
            </a>
          </div>
        </div>

        <div className="heroArtifact productPhone" aria-hidden="true">
          <div className="phoneTop">
            <strong>JetKZu Dispatch</strong>
            <span>Live route</span>
          </div>
          <div className="dispatchMap">
            <span className="mapLine" />
            <span className="mapNode pickup" />
            <span className="mapNode driver" />
            <span className="mapNode dropoff" />
          </div>
          <div className="dispatchPanel">
            <strong>Kabanbay Batyr Ave to Expo district</strong>
            <span>Driver assigned after passenger confirmation.</span>
          </div>
        </div>
      </section>

      <section className="platformGrid">
        <article className="surface">
          <div className="surfaceHeader">
            <CarProfile weight="bold" />
            <div>
              <span>Driver supply</span>
              <h2>Profiles and live status</h2>
            </div>
          </div>
          <p className="muted">Register drivers, set availability, and publish locations from the dashboard.</p>
        </article>
        <article className="surface">
          <div className="surfaceHeader">
            <CreditCard weight="bold" />
            <div>
              <span>Payments</span>
              <h2>Charge and receipt flow</h2>
            </div>
          </div>
          <p className="muted">Create payments, process a card charge, and open receipts for completed trips.</p>
        </article>
        <article className="surface">
          <div className="surfaceHeader">
            <ShieldCheck weight="bold" />
            <div>
              <span>Operations</span>
              <h2>Health and metrics</h2>
            </div>
          </div>
          <p className="muted">Gateway health, Prometheus scraping, and activity history stay visible locally.</p>
        </article>
      </section>
    </div>
  );
}
