import { Bell, CarProfile, CreditCard, MapPinArea, NavigationArrow, ShieldCheck } from "@phosphor-icons/react";
import type { Session } from "../types";
import { StatusBadge, Surface } from "../components/ui";

export function LandingPage({ session }: { session: Session }) {
  const dashboardHref = session.user ? "/dashboard" : "/login";

  return (
    <div className="landingLayout">
      <section className="hero reveal">
        <div className="heroCopy">
          <StatusBadge tone="warning">Astana taxi</StatusBadge>
          <h2>Book a car, follow the route, keep the receipt.</h2>
          <p>
            JetKZu is a compact taxi product for city rides. Choose pickup and dropoff points, request a driver, pay after the trip, and keep ride messages in your account.
          </p>
          <div className="heroActions">
            <a className="buttonLink" href={dashboardHref}>Request a ride</a>
            <a className="buttonLink secondaryButton" href="/login">Login</a>
          </div>
        </div>

        <div className="heroArtifact productPhone" aria-label="Ride booking preview">
          <div className="phoneTop">
            <span>Current ride</span>
            <strong>7 min</strong>
          </div>
          <div className="rideMap">
            <span className="mapNode pickup" />
            <span className="mapNode driver" />
            <span className="mapNode dropoff" />
            <div className="mapLine" />
          </div>
          <div className="rideSummary">
            <div>
              <span>Pickup</span>
              <strong>Kabanbay Batyr Ave</strong>
            </div>
            <div>
              <span>Dropoff</span>
              <strong>Expo business district</strong>
            </div>
            <div className="fareLine">
              <span>Estimated fare</span>
              <strong>820.40 KZT</strong>
            </div>
          </div>
        </div>
      </section>

      <section className="customerGrid">
        <Surface title="Ride booking" eyebrow="Passenger" icon={<MapPinArea weight="bold" />}>
          <p className="muted">Set route details, estimate the fare, and request a nearby driver from your account.</p>
        </Surface>
        <Surface title="Driver matching" eyebrow="Dispatch" icon={<CarProfile weight="bold" />}>
          <p className="muted">Drivers publish availability and location so the trip can move from request to pickup.</p>
        </Surface>
        <Surface title="Card payments" eyebrow="Billing" icon={<CreditCard weight="bold" />}>
          <p className="muted">Create payment, process the trip amount, and open the receipt after the ride.</p>
        </Surface>
        <Surface title="Ride messages" eyebrow="Inbox" icon={<Bell weight="bold" />}>
          <p className="muted">Receive ride updates and keep read history inside the dashboard.</p>
        </Surface>
      </section>

      <section className="trustBand reveal">
        <div>
          <ShieldCheck weight="bold" />
          <strong>Dashboard access is private.</strong>
          <p>Everyone can view this landing page. Ride history, payments, and notifications are available only after login.</p>
        </div>
        <a className="buttonLink secondaryButton" href={dashboardHref}>
          <NavigationArrow weight="bold" />
          <span>{session.user ? "Open dashboard" : "Continue to login"}</span>
        </a>
      </section>
    </div>
  );
}
