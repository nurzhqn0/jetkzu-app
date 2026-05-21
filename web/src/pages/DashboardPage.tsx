import { Bell, CarProfile, CheckCircle, CreditCard, MapPinArea, NavigationArrow, Pulse, Receipt } from "@phosphor-icons/react";
import { useState } from "react";
import { api, type ApiResult } from "../api";
import { ActionButton, Field, Metric, StatusBadge, Surface } from "../components/ui";
import type { PageProps } from "../types";

type DashboardErrors = Partial<Record<"driverId" | "rideId" | "paymentId" | "notificationId" | "pickup" | "dropoff", string>>;

const idFrom = (result: ApiResult, keys: string[]) => {
  if (!result.ok || !result.body || typeof result.body !== "object") return "";
  const body = result.body as Record<string, unknown>;
  for (const key of keys) {
    const value = body[key];
    if (typeof value === "string" && value.trim()) return value;
  }
  return "";
};

const amountFrom = (result: ApiResult) => {
  if (!result.ok || !result.body || typeof result.body !== "object") return "";
  const body = result.body as Record<string, unknown>;
  const value = body.estimated_fare ?? body.amount ?? body.total;
  return typeof value === "number" ? `${value.toFixed(2)} KZT` : "";
};

export function DashboardPage({ token, session, record }: PageProps) {
  const [pickup, setPickup] = useState("Kabanbay Batyr Ave");
  const [dropoff, setDropoff] = useState("Expo business district");
  const [driverId, setDriverId] = useState("");
  const [rideId, setRideId] = useState("");
  const [paymentId, setPaymentId] = useState("");
  const [notificationId, setNotificationId] = useState("");
  const [estimate, setEstimate] = useState("Not calculated");
  const [rideStatus, setRideStatus] = useState("Planning");
  const [driverStatus, setDriverStatus] = useState("No driver yet");
  const [paymentStatus, setPaymentStatus] = useState("Not charged");
  const [inboxStatus, setInboxStatus] = useState("No messages opened");
  const [errors, setErrors] = useState<DashboardErrors>({});
  const [loadingAction, setLoadingAction] = useState("");
  const userId = session.user?.id ?? "";

  const run = async (name: string, action: () => Promise<void>) => {
    setLoadingAction(name);
    setErrors({});
    try {
      await action();
    } finally {
      setLoadingAction("");
    }
  };

  const requireValue = (key: keyof DashboardErrors, value: string, message: string) => {
    if (value.trim()) return true;
    setErrors((current) => ({ ...current, [key]: message }));
    return false;
  };

  const rideBody = {
    passenger_id: userId,
    pickup_address: pickup,
    dropoff_address: dropoff,
    pickup_lat: 51.169,
    pickup_lng: 71.449,
    dropoff_lat: 51.18,
    dropoff_lng: 71.46
  };

  const estimateRide = () => run("estimate", async () => {
    if (!requireValue("pickup", pickup, "Enter pickup address.") || !requireValue("dropoff", dropoff, "Enter destination.")) return;
    const result = await api("/api/rides/estimate", { method: "POST", body: JSON.stringify(rideBody) }, token);
    record("Fare estimated", result);
    setEstimate(amountFrom(result) || "820.40 KZT");
  });

  const requestRide = () => run("request", async () => {
    if (!requireValue("pickup", pickup, "Enter pickup address.") || !requireValue("dropoff", dropoff, "Enter destination.")) return;
    const result = await api("/api/rides", { method: "POST", body: JSON.stringify(rideBody) }, token);
    record("Ride requested", result);
    const nextRideId = idFrom(result, ["ride_id", "id"]);
    if (nextRideId) setRideId(nextRideId);
    if (result.ok) setRideStatus("Searching for driver");
  });

  const completeRide = () => run("complete", async () => {
    if (!requireValue("rideId", rideId, "Enter the ride ID to complete.")) return;
    const result = await api(`/api/rides/${rideId}/complete`, { method: "POST" }, token);
    record("Ride completed", result);
    if (result.ok) setRideStatus("Completed");
  });

  const registerDriver = () => run("driver-register", async () => {
    const result = await api("/api/drivers/register", { method: "POST", body: JSON.stringify({ user_id: userId, license_number: "KZ-2026-7788" }) }, token);
    record("Driver profile created", result);
    const nextDriverId = idFrom(result, ["driver_id", "id"]);
    if (nextDriverId) setDriverId(nextDriverId);
    if (result.ok) setDriverStatus("Driver profile ready");
  });

  const goOnline = () => run("driver-online", async () => {
    if (!requireValue("driverId", driverId, "Enter the driver ID first.")) return;
    const result = await api("/api/drivers/status", { method: "PATCH", body: JSON.stringify({ driver_id: driverId, status: "online" }) }, token);
    record("Driver set online", result);
    if (result.ok) setDriverStatus("Available for dispatch");
  });

  const publishLocation = () => run("driver-location", async () => {
    if (!requireValue("driverId", driverId, "Enter the driver ID first.")) return;
    const result = await api("/api/drivers/location", { method: "PATCH", body: JSON.stringify({ driver_id: driverId, latitude: 51.169, longitude: 71.449 }) }, token);
    record("Driver location updated", result);
    if (result.ok) setDriverStatus("Location is live");
  });

  const createPayment = () => run("payment-create", async () => {
    if (!requireValue("rideId", rideId, "Request or enter a ride ID before payment.")) return;
    const result = await api("/api/payments", { method: "POST", body: JSON.stringify({ ride_id: rideId, user_id: userId, amount: 820.4, method: "card" }) }, token);
    record("Payment created", result);
    const nextPaymentId = idFrom(result, ["payment_id", "id"]);
    if (nextPaymentId) setPaymentId(nextPaymentId);
    if (result.ok) setPaymentStatus("Ready to charge");
  });

  const processPayment = () => run("payment-process", async () => {
    if (!requireValue("paymentId", paymentId, "Enter the payment ID to charge.")) return;
    const result = await api("/api/payments/process", { method: "POST", body: JSON.stringify({ payment_id: paymentId }) }, token);
    record("Payment charged", result);
    if (result.ok) setPaymentStatus("Paid");
  });

  const receipt = () => run("receipt", async () => {
    if (!requireValue("paymentId", paymentId, "Enter the payment ID to open receipt.")) return;
    const result = await api(`/api/payments/${paymentId}/receipt`, {}, token);
    record("Receipt opened", result);
    if (result.ok) setPaymentStatus("Receipt available");
  });

  const inbox = () => run("inbox", async () => {
    const result = await api("/api/notifications/my", {}, token);
    record("Inbox opened", result);
    if (result.ok) setInboxStatus("Inbox refreshed");
  });

  const sendNotification = () => run("send-notification", async () => {
    const result = await api("/api/notifications/email", { method: "POST", body: JSON.stringify({ user_id: userId, to: session.user?.email, subject: "JetKZu ride update", body: "Your ride status changed." }) }, token);
    record("Ride update sent", result);
    const nextNotificationId = idFrom(result, ["notification_id", "id"]);
    if (nextNotificationId) setNotificationId(nextNotificationId);
    if (result.ok) setInboxStatus("Ride update sent");
  });

  const markRead = () => run("mark-read", async () => {
    if (!requireValue("notificationId", notificationId, "Enter notification ID to mark read.")) return;
    const result = await api(`/api/notifications/${notificationId}/read`, { method: "PATCH" }, token);
    record("Message marked read", result);
    if (result.ok) setInboxStatus("Message marked read");
  });

  return (
    <div className="dashboard">
      <div className="metricGrid reveal">
        <Metric label="Ride" value={rideStatus} />
        <Metric label="Driver" value={driverStatus} />
        <Metric label="Fare" value={estimate} />
        <Metric label="Payment" value={paymentStatus} />
      </div>

      <div className="dashboardGrid">
        <Surface title="Plan your ride" eyebrow="Booking" icon={<MapPinArea weight="bold" />} wide>
          <div className="routeFields">
            <Field label="Pickup" value={pickup} onChange={setPickup} error={errors.pickup} helper="Where the driver should meet you." />
            <Field label="Destination" value={dropoff} onChange={setDropoff} error={errors.dropoff} helper="Where you want to arrive." />
          </div>
          <div className="routePreview">
            <StatusBadge tone="blue">Route</StatusBadge>
            <strong>{pickup} to {dropoff}</strong>
            <span>JetKZu estimates the route and stores the requested ride in your account.</span>
          </div>
          <Field label="Ride ID" value={rideId} onChange={setRideId} error={errors.rideId} helper="Filled after request if the backend returns an ID." />
          <div className="actions">
            <ActionButton icon={<NavigationArrow weight="bold" />} label="Estimate fare" onClick={estimateRide} loading={loadingAction === "estimate"} variant="secondary" />
            <ActionButton icon={<MapPinArea weight="bold" />} label="Request driver" onClick={requestRide} loading={loadingAction === "request"} />
            <ActionButton icon={<CheckCircle weight="bold" />} label="Complete ride" onClick={completeRide} loading={loadingAction === "complete"} />
          </div>
        </Surface>

        <Surface title="Driver mode" eyebrow="Supply" icon={<CarProfile weight="bold" />}>
          <p className="muted">Use this when the signed-in user is joining as a driver or testing dispatch supply.</p>
          <Field label="Driver ID" value={driverId} onChange={setDriverId} error={errors.driverId} helper="Filled after driver profile creation when available." />
          <div className="actions">
            <ActionButton icon={<CarProfile weight="bold" />} label="Create driver profile" onClick={registerDriver} loading={loadingAction === "driver-register"} />
            <ActionButton icon={<Pulse weight="bold" />} label="Go online" onClick={goOnline} loading={loadingAction === "driver-online"} variant="secondary" />
            <ActionButton icon={<NavigationArrow weight="bold" />} label="Share location" onClick={publishLocation} loading={loadingAction === "driver-location"} variant="secondary" />
          </div>
        </Surface>

        <Surface title="Payment and receipt" eyebrow="Wallet" icon={<CreditCard weight="bold" />}>
          <Field label="Payment ID" value={paymentId} onChange={setPaymentId} error={errors.paymentId} helper="Filled after payment creation when available." />
          <div className="actions">
            <ActionButton icon={<CreditCard weight="bold" />} label="Create payment" onClick={createPayment} loading={loadingAction === "payment-create"} />
            <ActionButton icon={<CheckCircle weight="bold" />} label="Charge card" onClick={processPayment} loading={loadingAction === "payment-process"} />
            <ActionButton icon={<Receipt weight="bold" />} label="Open receipt" onClick={receipt} loading={loadingAction === "receipt"} variant="secondary" />
          </div>
        </Surface>

        <Surface title="Ride inbox" eyebrow="Notifications" icon={<Bell weight="bold" />}>
          <div className="routePreview">
            <StatusBadge tone="success">Messages</StatusBadge>
            <strong>{inboxStatus}</strong>
            <span>Open ride updates, send the latest ride status, or mark a message as read.</span>
          </div>
          <Field label="Notification ID" value={notificationId} onChange={setNotificationId} error={errors.notificationId} helper="Filled after sending when the backend returns an ID." />
          <div className="actions">
            <ActionButton icon={<Bell weight="bold" />} label="Open inbox" onClick={inbox} loading={loadingAction === "inbox"} variant="secondary" />
            <ActionButton icon={<Bell weight="bold" />} label="Send ride update" onClick={sendNotification} loading={loadingAction === "send-notification"} />
            <ActionButton icon={<CheckCircle weight="bold" />} label="Mark read" onClick={markRead} loading={loadingAction === "mark-read"} variant="secondary" />
          </div>
        </Surface>
      </div>
    </div>
  );
}
