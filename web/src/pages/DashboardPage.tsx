import {
  Bell,
  CarProfile,
  CheckCircle,
  Clock,
  CreditCard,
  CurrencyCircleDollar,
  MapPinArea,
  NavigationArrow,
  Receipt,
  SteeringWheel,
  Wallet,
  WarningCircle
} from "@phosphor-icons/react";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { api, type ApiResult } from "../api";
import { ActionButton, Metric, StatusBadge, Surface } from "../components/ui";
import type { AppRole, PageProps } from "../types";

type PaymentMethod = "cash" | "card";
type ToastTone = "success" | "warning" | "blue";

type Place = {
  id: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
};

type Ride = {
  id: string;
  passenger_id: string;
  driver_id: string;
  pickup_lat: number;
  pickup_lng: number;
  dropoff_lat: number;
  dropoff_lng: number;
  status: string;
  price: number;
};

type Payment = {
  id: string;
  status: string;
  amount: number;
  method: string;
};

type RideMeta = {
  rideId: string;
  pickup: string;
  dropoff: string;
  passengerName: string;
  paymentMethod: PaymentMethod;
  cardLast4?: string;
  paid?: boolean;
  paymentId?: string;
  createdAt: string;
};

type DriverProfile = {
  id: string;
  status: string;
};

type Toast = {
  id: string;
  title: string;
  message: string;
  tone: ToastTone;
};

const PLACES: Place[] = [
  { id: "mega", name: "MEGA Silk Way", address: "проспект Кабанбай Батыра, 62", lat: 51.0897, lng: 71.4109 },
  { id: "expo", name: "EXPO", address: "проспект Мәңгілік Ел, 53/1", lat: 51.0903, lng: 71.4186 },
  { id: "baiterek", name: "Байтерек", address: "бульвар Нуржол, 14", lat: 51.1282, lng: 71.4304 },
  { id: "airport", name: "Аэропорт Астана", address: "проспект Кабанбай Батыра", lat: 51.0222, lng: 71.4669 },
  { id: "railway", name: "Нурлы Жол", address: "железнодорожный вокзал Нурлы Жол", lat: 51.0442, lng: 71.5222 },
  { id: "keruen", name: "Keruen City", address: "улица Сыганак, 16", lat: 51.1172, lng: 71.4276 }
];

const ACTIVE_STATUSES = new Set(["requested", "driver_assigned", "driver_arrived", "in_progress"]);
const RIDE_META_STORAGE = "jetkzu-ride-meta";

const asRecord = (value: unknown): Record<string, unknown> | null =>
  value && typeof value === "object" ? (value as Record<string, unknown>) : null;

const stringOf = (record: Record<string, unknown>, key: string, fallback = "") => {
  const value = record[key];
  return typeof value === "string" ? value : fallback;
};

const numberOf = (record: Record<string, unknown>, key: string, fallback = 0) => {
  const value = record[key];
  return typeof value === "number" && Number.isFinite(value) ? value : fallback;
};

const rideFromUnknown = (value: unknown): Ride | null => {
  const record = asRecord(value);
  if (!record) return null;
  return {
    id: stringOf(record, "id"),
    passenger_id: stringOf(record, "passenger_id"),
    driver_id: stringOf(record, "driver_id"),
    pickup_lat: numberOf(record, "pickup_lat"),
    pickup_lng: numberOf(record, "pickup_lng"),
    dropoff_lat: numberOf(record, "dropoff_lat"),
    dropoff_lng: numberOf(record, "dropoff_lng"),
    status: stringOf(record, "status", "requested"),
    price: numberOf(record, "price")
  };
};

const rideFromBody = (body: unknown): Ride | null => {
  const record = asRecord(body);
  if (!record) return null;
  return rideFromUnknown(record.ride) ?? rideFromUnknown(body);
};

const ridesFromBody = (body: unknown): Ride[] => {
  const record = asRecord(body);
  const raw = record?.rides;
  if (!Array.isArray(raw)) return [];
  return raw.map(rideFromUnknown).filter((ride): ride is Ride => Boolean(ride?.id));
};

const paymentFromBody = (body: unknown): Payment | null => {
  const record = asRecord(body);
  const raw = asRecord(record?.payment ?? body);
  if (!raw) return null;
  return {
    id: stringOf(raw, "id"),
    status: stringOf(raw, "status"),
    amount: numberOf(raw, "amount"),
    method: stringOf(raw, "method")
  };
};

const driverFromBody = (body: unknown): DriverProfile | null => {
  const record = asRecord(body);
  const raw = asRecord(record?.driver ?? body);
  if (!raw) return null;
  const id = stringOf(raw, "id");
  return id ? { id, status: stringOf(raw, "status", "offline") } : null;
};

const driversFromBody = (body: unknown) => {
  const record = asRecord(body);
  const raw = record?.drivers;
  return Array.isArray(raw) ? raw.map(asRecord).filter((driver): driver is Record<string, unknown> => Boolean(driver)) : [];
};

const readRideMetas = (): Record<string, RideMeta> => {
  try {
    const raw = localStorage.getItem(RIDE_META_STORAGE);
    return raw ? JSON.parse(raw) as Record<string, RideMeta> : {};
  } catch {
    return {};
  }
};

const writeRideMetas = (metas: Record<string, RideMeta>) => {
  localStorage.setItem(RIDE_META_STORAGE, JSON.stringify(metas));
};

const driverStorageKey = (userId: string) => `jetkzu-driver-${userId}`;
const balanceStorageKey = (userId: string) => `jetkzu-driver-balance-${userId}`;

const readDriverProfile = (userId: string): DriverProfile | null => {
  if (!userId) return null;
  try {
    const raw = localStorage.getItem(driverStorageKey(userId));
    return raw ? JSON.parse(raw) as DriverProfile : null;
  } catch {
    return null;
  }
};

const writeDriverProfile = (userId: string, profile: DriverProfile) => {
  localStorage.setItem(driverStorageKey(userId), JSON.stringify(profile));
};

const formatMoney = (amount: number) => `${Math.round(amount).toLocaleString("ru-RU")} KZT`;
const isActiveRide = (ride: Ride) => ACTIVE_STATUSES.has(ride.status);
const selectedPlace = (id: string) => PLACES.find((place) => place.id === id) ?? PLACES[0];
const placeLabel = (place: Place) => `${place.name}, ${place.address}`;

const statusText: Record<string, string> = {
  requested: "Ждём таксиста",
  driver_assigned: "Таксист принял заказ",
  driver_arrived: "Таксист приехал",
  in_progress: "В пути",
  completed: "Поездка завершена",
  cancelled: "Заказ отменён"
};

const statusTone = (status: string): ToastTone => {
  if (status === "completed") return "success";
  if (status === "cancelled") return "warning";
  return "blue";
};

export function DashboardPage({ token, session, record }: PageProps) {
  const userId = session.user?.id ?? "";
  const role: AppRole = session.app_role ?? (session.user?.role === "driver" ? "driver" : "passenger");

  const [pickupId, setPickupId] = useState("baiterek");
  const [dropoffId, setDropoffId] = useState("expo");
  const [paymentMethod, setPaymentMethod] = useState<PaymentMethod>("cash");
  const [cardNumber, setCardNumber] = useState("4400 0000 0000 2026");
  const [cardExpiry, setCardExpiry] = useState("12/28");
  const [cardCvc, setCardCvc] = useState("777");
  const [estimate, setEstimate] = useState<{ price: number; distance: number } | null>(null);
  const [activeRide, setActiveRide] = useState<Ride | null>(null);
  const [lastRide, setLastRide] = useState<Ride | null>(null);
  const [rideMetas, setRideMetas] = useState<Record<string, RideMeta>>(() => readRideMetas());
  const [driverProfile, setDriverProfile] = useState<DriverProfile | null>(() => readDriverProfile(userId));
  const [waitingRides, setWaitingRides] = useState<Ride[]>([]);
  const [currentDriverRide, setCurrentDriverRide] = useState<Ride | null>(null);
  const [driverBalance, setDriverBalance] = useState(() => Number(localStorage.getItem(balanceStorageKey(userId)) ?? "0"));
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [loadingAction, setLoadingAction] = useState("");
  const seenStatuses = useRef<Record<string, string>>({});

  const pickup = selectedPlace(pickupId);
  const dropoff = selectedPlace(dropoffId);
  const activeMeta = activeRide ? rideMetas[activeRide.id] : null;
  const currentDriverMeta = currentDriverRide ? rideMetas[currentDriverRide.id] : null;

  const metrics = useMemo(() => {
    if (role === "driver") {
      return [
        { label: "Роль", value: "Таксист" },
        { label: "Заказы", value: String(waitingRides.length) },
        { label: "Поездка", value: currentDriverRide ? statusText[currentDriverRide.status] ?? currentDriverRide.status : "Нет активной" },
        { label: "Баланс", value: formatMoney(driverBalance) }
      ];
    }
    return [
      { label: "Роль", value: "Клиент" },
      { label: "Заказ", value: activeRide ? statusText[activeRide.status] ?? activeRide.status : "Нет активного" },
      { label: "Тариф", value: estimate ? formatMoney(estimate.price) : "Не рассчитан" },
      { label: "Оплата", value: paymentMethod === "card" ? "Карта" : "Наличка" }
    ];
  }, [activeRide, currentDriverRide, driverBalance, estimate, paymentMethod, role, waitingRides.length]);

  const notify = (title: string, message: string, tone: ToastTone = "blue") => {
    const id = crypto.randomUUID();
    setToasts((items) => [{ id, title, message, tone }, ...items].slice(0, 5));
    window.setTimeout(() => setToasts((items) => items.filter((item) => item.id !== id)), 5600);
  };

  const run = async (name: string, action: () => Promise<void>) => {
    setLoadingAction(name);
    try {
      await action();
    } catch (error) {
      notify("Что-то пошло не так", error instanceof Error ? error.message : "Действие не выполнено.", "warning");
    } finally {
      setLoadingAction("");
    }
  };

  const saveRideMeta = (rideId: string, patch: Partial<RideMeta>) => {
    setRideMetas((current) => {
      const existing = current[rideId];
      const next = {
        ...current,
        [rideId]: {
          rideId,
          pickup: existing?.pickup ?? placeLabel(pickup),
          dropoff: existing?.dropoff ?? placeLabel(dropoff),
          passengerName: existing?.passengerName ?? session.user?.full_name ?? "Клиент",
          paymentMethod: existing?.paymentMethod ?? paymentMethod,
          createdAt: existing?.createdAt ?? new Date().toISOString(),
          ...patch
        }
      };
      writeRideMetas(next);
      return next;
    });
  };

  const rememberStatus = (ride: Ride) => {
    const previous = seenStatuses.current[ride.id];
    if (previous && previous !== ride.status) {
      notify(statusText[ride.status] ?? "Статус изменился", statusMessage(ride, rideMetas[ride.id]), statusTone(ride.status));
    }
    seenStatuses.current[ride.id] = ride.status;
  };

  const estimateRide = async () => {
    if (pickupId === dropoffId) {
      notify("Маршрут не выбран", "Точка посадки и пункт назначения должны отличаться.", "warning");
      return;
    }
    const result = await api("/api/rides/estimate", {
      method: "POST",
      body: JSON.stringify({
        pickup_lat: pickup.lat,
        pickup_lng: pickup.lng,
        dropoff_lat: dropoff.lat,
        dropoff_lng: dropoff.lng
      })
    }, token);
    record("Расчёт стоимости", result);
    if (!result.ok) {
      notify("Не удалось рассчитать", "Проверьте, что ride-service доступен.", "warning");
      return;
    }
    const body = asRecord(result.body);
    setEstimate({
      price: numberOf(body ?? {}, "price", 700),
      distance: numberOf(body ?? {}, "distance_km")
    });
    notify("Цена рассчитана", `${placeLabel(pickup)} → ${placeLabel(dropoff)}`, "success");
  };

  const validateCard = () => {
    const digits = cardNumber.replace(/\D/g, "");
    if (paymentMethod === "cash") return true;
    if (digits.length < 12 || cardExpiry.trim().length < 4 || cardCvc.replace(/\D/g, "").length < 3) {
      notify("Проверьте карту", "Для демо-оплаты нужны номер карты, срок и CVC.", "warning");
      return false;
    }
    return true;
  };

  const requestRide = () => run("request-ride", async () => {
    if (activeRide && isActiveRide(activeRide)) {
      notify("У вас уже есть заказ", "Завершите или отмените текущую поездку перед новым заказом.", "warning");
      return;
    }
    if (!validateCard()) return;
    if (!estimate) await estimateRide();

    const result = await api("/api/rides", {
      method: "POST",
      body: JSON.stringify({
        pickup_lat: pickup.lat,
        pickup_lng: pickup.lng,
        dropoff_lat: dropoff.lat,
        dropoff_lng: dropoff.lng
      })
    }, token);
    record("Заказ такси", result);
    const ride = rideFromBody(result.body);
    if (!result.ok || !ride) {
      notify("Заказ не создан", "Backend не вернул поездку. Попробуйте ещё раз.", "warning");
      return;
    }

    setActiveRide(ride);
    setLastRide(ride);
    seenStatuses.current[ride.id] = ride.status;
    saveRideMeta(ride.id, {
      pickup: placeLabel(pickup),
      dropoff: placeLabel(dropoff),
      passengerName: session.user?.full_name ?? "Клиент",
      paymentMethod,
      cardLast4: paymentMethod === "card" ? cardNumber.replace(/\D/g, "").slice(-4) : undefined,
      paid: false
    });
    notify("Заказ создан", "Ждём таксиста. Когда он примет заказ, статус обновится.", "success");
  });

  const cancelRide = () => run("cancel-ride", async () => {
    if (!activeRide) return;
    const result = await api(`/api/rides/${activeRide.id}/cancel`, {
      method: "POST",
      body: JSON.stringify({ reason: "cancelled by passenger" })
    }, token);
    record("Отмена заказа", result);
    const ride = rideFromBody(result.body);
    if (result.ok && ride) {
      setActiveRide(null);
      setLastRide(ride);
      seenStatuses.current[ride.id] = ride.status;
      notify("Заказ отменён", "Активных заказов больше нет.", "warning");
    }
  });

  const refreshPassengerRides = async () => {
    const result = await api("/api/rides/my", {}, token);
    if (!result.ok) return;
    const rides = ridesFromBody(result.body);
    const active = rides.find(isActiveRide) ?? null;
    const latest = rides[0] ?? null;
    if (active) rememberStatus(active);
    setActiveRide(active);
    setLastRide(latest);
  };

  const findExistingDriver = async (): Promise<DriverProfile | null> => {
    const result = await api("/api/drivers?limit=100", {}, token);
    if (!result.ok) return null;
    const found = driversFromBody(result.body).find((driver) => stringOf(driver, "user_id") === userId);
    if (!found) return null;
    const id = stringOf(found, "id");
    if (!id) return null;
    const profile = { id, status: stringOf(found, "status", "offline") };
    setDriverProfile(profile);
    writeDriverProfile(userId, profile);
    return profile;
  };

  const ensureDriverProfile = async (): Promise<DriverProfile | null> => {
    if (driverProfile?.id) return driverProfile;
    const existing = await findExistingDriver();
    if (existing) return existing;

    const phoneDigits = (session.user?.phone ?? session.user?.email ?? "000000").replace(/\D/g, "");
    const result = await api("/api/drivers/register", {
      method: "POST",
      body: JSON.stringify({ license_number: `KZ-${phoneDigits.slice(-6).padStart(6, "0")}` })
    }, token);
    record("Профиль таксиста", result);
    const profile = driverFromBody(result.body);
    if (!result.ok || !profile) {
      notify("Профиль таксиста не создан", "Попробуйте обновить смену или проверьте driver-service.", "warning");
      return null;
    }

    writeDriverProfile(userId, profile);
    setDriverProfile(profile);
    await api("/api/drivers/vehicle", {
      method: "POST",
      body: JSON.stringify({
        driver_id: profile.id,
        plate_number: `JET-${phoneDigits.slice(-3).padStart(3, "0")}`,
        make: "Hyundai",
        model: "Elantra",
        year: 2022,
        color: "white"
      })
    }, token);
    return profile;
  };

  const refreshDriverData = async (profile = driverProfile) => {
    const result = await api("/api/rides/active?limit=50", {}, token);
    if (!result.ok) return;
    const rides = ridesFromBody(result.body);
    setWaitingRides(rides.filter((ride) => ride.status === "requested"));
    if (profile?.id) {
      const current = rides.find((ride) => ride.driver_id === profile.id && isActiveRide(ride)) ?? null;
      if (current) rememberStatus(current);
      setCurrentDriverRide(current);
    }
  };

  const startShift = () => run("start-shift", async () => {
    const profile = await ensureDriverProfile();
    if (!profile) return;
    const result = await api("/api/drivers/status", {
      method: "PATCH",
      body: JSON.stringify({ driver_id: profile.id, status: "online" })
    }, token);
    record("Смена таксиста", result);
    if (result.ok) {
      const next = { ...profile, status: "online" };
      setDriverProfile(next);
      writeDriverProfile(userId, next);
      notify("Вы на линии", "Список заказов обновлён. Можно принять подходящий заказ.", "success");
    }
    await refreshDriverData(profile);
  });

  const acceptRide = (ride: Ride) => run(`accept-${ride.id}`, async () => {
    const profile = await ensureDriverProfile();
    if (!profile) return;
    const result = await api(`/api/rides/${ride.id}/accept`, {
      method: "POST",
      body: JSON.stringify({ driver_id: profile.id })
    }, token);
    record("Заказ принят", result);
    const accepted = rideFromBody(result.body) ?? { ...ride, driver_id: profile.id, status: "driver_assigned" };
    if (!result.ok) {
      notify("Заказ уже недоступен", "Его мог принять другой таксист или статус изменился.", "warning");
      await refreshDriverData(profile);
      return;
    }
    setCurrentDriverRide(accepted);
    setWaitingRides((rides) => rides.filter((item) => item.id !== ride.id));
    seenStatuses.current[ride.id] = accepted.status;
    await api("/api/drivers/status", {
      method: "PATCH",
      body: JSON.stringify({ driver_id: profile.id, status: "busy" })
    }, token);
    notify("Заказ принят", "Клиент увидит, что таксист едет к нему.", "success");
  });

  const updateDriverRideStatus = (status: "driver_arrived" | "in_progress" | "completed") => run(`ride-${status}`, async () => {
    if (!currentDriverRide) return;
    const result: ApiResult = status === "completed"
      ? await api(`/api/rides/${currentDriverRide.id}/complete`, { method: "POST" }, token)
      : await api(`/api/rides/${currentDriverRide.id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status, reason: statusText[status] ?? status })
      }, token);

    record(statusText[status] ?? "Статус поездки", result);
    const ride = rideFromBody(result.body) ?? { ...currentDriverRide, status };
    if (!result.ok) {
      notify("Статус не обновился", "Проверьте последовательность: приехал → в пути → завершить.", "warning");
      return;
    }

    setCurrentDriverRide(status === "completed" ? null : ride);
    setLastRide(ride);
    seenStatuses.current[ride.id] = ride.status;
    notify(statusText[status], statusMessage(ride, rideMetas[ride.id]), statusTone(status));

    if (status === "in_progress" && rideMetas[ride.id]?.paymentMethod === "card") {
      await settleCardPayment(ride);
    }
    if (status === "completed") {
      if (rideMetas[ride.id]?.paymentMethod === "cash") {
        notify("Оплата наличными", `К получению: ${formatMoney(ride.price)}.`, "success");
      }
      if (driverProfile?.id) {
        await api("/api/drivers/status", {
          method: "PATCH",
          body: JSON.stringify({ driver_id: driverProfile.id, status: "online" })
        }, token);
      }
      await refreshDriverData(driverProfile);
    }
  });

  const settleCardPayment = async (ride: Ride) => {
    const meta = rideMetas[ride.id];
    if (meta?.paid) return;

    await new Promise((resolve) => window.setTimeout(resolve, 500));
    let payment: Payment | null = null;
    const existing = await api(`/api/rides/${ride.id}/payment`, {}, token);
    if (existing.ok) payment = paymentFromBody(existing.body);

    if (!payment?.id) {
      const created = await api("/api/payments", {
        method: "POST",
        body: JSON.stringify({
          ride_id: ride.id,
          user_id: ride.passenger_id,
          amount: ride.price,
          method: "card"
        })
      }, token);
      record("Демо-оплата создана", created);
      payment = paymentFromBody(created.body);
    }

    if (payment?.id && payment.status !== "succeeded") {
      const processed = await api("/api/payments/process", {
        method: "POST",
        body: JSON.stringify({ payment_id: payment.id })
      }, token);
      record("Карта оплачена", processed);
      payment = paymentFromBody(processed.body) ?? payment;
    }

    if (payment?.id) {
      saveRideMeta(ride.id, { paid: true, paymentId: payment.id });
      const nextBalance = driverBalance + ride.price;
      setDriverBalance(nextBalance);
      localStorage.setItem(balanceStorageKey(userId), String(nextBalance));
      notify("Баланс пополнен", `Карта клиента оплачена: ${formatMoney(ride.price)}.`, "success");
    }
  };

  useEffect(() => {
    setDriverProfile(readDriverProfile(userId));
    setDriverBalance(Number(localStorage.getItem(balanceStorageKey(userId)) ?? "0"));
  }, [userId]);

  useEffect(() => {
    if (!token) return undefined;
    let stopped = false;
    const load = async () => {
      if (stopped) return;
      if (role === "passenger") {
        await refreshPassengerRides();
      } else {
        const profile = driverProfile ?? await findExistingDriver();
        await refreshDriverData(profile);
      }
    };
    void load();
    const timer = window.setInterval(() => { void load(); }, 5000);
    return () => {
      stopped = true;
      window.clearInterval(timer);
    };
  }, [token, role, driverProfile?.id]);

  return (
    <div className="dashboard taxiWorkspace">
      <ToastStack toasts={toasts} />

      <div className="metricGrid reveal">
        {metrics.map((metric) => <Metric key={metric.label} label={metric.label} value={metric.value} />)}
      </div>

      {role === "passenger" ? (
        <PassengerView
          pickupId={pickupId}
          dropoffId={dropoffId}
          paymentMethod={paymentMethod}
          cardNumber={cardNumber}
          cardExpiry={cardExpiry}
          cardCvc={cardCvc}
          estimate={estimate}
          activeRide={activeRide}
          lastRide={lastRide}
          activeMeta={activeMeta}
          loadingAction={loadingAction}
          onPickup={setPickupId}
          onDropoff={setDropoffId}
          onPaymentMethod={setPaymentMethod}
          onCardNumber={setCardNumber}
          onCardExpiry={setCardExpiry}
          onCardCvc={setCardCvc}
          onEstimate={() => { void run("estimate", estimateRide); }}
          onRequest={requestRide}
          onCancel={cancelRide}
        />
      ) : (
        <DriverView
          profile={driverProfile}
          waitingRides={waitingRides}
          currentRide={currentDriverRide}
          currentMeta={currentDriverMeta}
          rideMetas={rideMetas}
          balance={driverBalance}
          loadingAction={loadingAction}
          onStartShift={startShift}
          onRefresh={() => { void run("refresh-driver", async () => refreshDriverData(driverProfile)); }}
          onAccept={acceptRide}
          onArrived={() => updateDriverRideStatus("driver_arrived")}
          onStartTrip={() => updateDriverRideStatus("in_progress")}
          onComplete={() => updateDriverRideStatus("completed")}
        />
      )}
    </div>
  );
}

function PassengerView({
  pickupId,
  dropoffId,
  paymentMethod,
  cardNumber,
  cardExpiry,
  cardCvc,
  estimate,
  activeRide,
  lastRide,
  activeMeta,
  loadingAction,
  onPickup,
  onDropoff,
  onPaymentMethod,
  onCardNumber,
  onCardExpiry,
  onCardCvc,
  onEstimate,
  onRequest,
  onCancel
}: {
  pickupId: string;
  dropoffId: string;
  paymentMethod: PaymentMethod;
  cardNumber: string;
  cardExpiry: string;
  cardCvc: string;
  estimate: { price: number; distance: number } | null;
  activeRide: Ride | null;
  lastRide: Ride | null;
  activeMeta: RideMeta | null;
  loadingAction: string;
  onPickup: (value: string) => void;
  onDropoff: (value: string) => void;
  onPaymentMethod: (value: PaymentMethod) => void;
  onCardNumber: (value: string) => void;
  onCardExpiry: (value: string) => void;
  onCardCvc: (value: string) => void;
  onEstimate: () => void;
  onRequest: () => void;
  onCancel: () => void;
}) {
  const ride = activeRide ?? lastRide;
  const canOrder = !activeRide || !isActiveRide(activeRide);

  return (
    <div className="dashboardGrid">
      <Surface title="Заказать такси" eyebrow="Клиент" icon={<MapPinArea weight="bold" />} wide>
        <div className="routeFields">
          <PlaceSelect label="Откуда" value={pickupId} onChange={onPickup} />
          <PlaceSelect label="Куда" value={dropoffId} onChange={onDropoff} />
        </div>

        <div className="routePreview">
          <StatusBadge tone="blue">Маршрут</StatusBadge>
          <strong>{placeLabel(selectedPlace(pickupId))} → {placeLabel(selectedPlace(dropoffId))}</strong>
          <span>{estimate ? `${formatMoney(estimate.price)} · ${estimate.distance.toFixed(1)} км` : "Нажмите расчёт, чтобы увидеть примерную цену."}</span>
        </div>

        <div className="paymentChoice" aria-label="Выбор оплаты">
          <PaymentOption active={paymentMethod === "cash"} title="Наличка" text="Оплата в конце поездки" icon={<Wallet weight="bold" />} onClick={() => onPaymentMethod("cash")} />
          <PaymentOption active={paymentMethod === "card"} title="Картой" text="Демо-списание в пути" icon={<CreditCard weight="bold" />} onClick={() => onPaymentMethod("card")} />
        </div>

        {paymentMethod === "card" && (
          <div className="cardFields">
            <label className="field">
              <span>Номер карты</span>
              <input value={cardNumber} onChange={(event) => onCardNumber(event.target.value)} inputMode="numeric" />
            </label>
            <label className="field">
              <span>Срок</span>
              <input value={cardExpiry} onChange={(event) => onCardExpiry(event.target.value)} placeholder="MM/YY" />
            </label>
            <label className="field">
              <span>CVC</span>
              <input value={cardCvc} onChange={(event) => onCardCvc(event.target.value)} inputMode="numeric" />
            </label>
          </div>
        )}

        {!canOrder && (
          <div className="inlineWarning">
            <WarningCircle weight="bold" />
            <span>Нельзя создать второй заказ, пока текущая поездка активна.</span>
          </div>
        )}

        <div className="actions">
          <ActionButton icon={<CurrencyCircleDollar weight="bold" />} label="Рассчитать" onClick={onEstimate} loading={loadingAction === "estimate"} variant="secondary" />
          <ActionButton icon={<NavigationArrow weight="bold" />} label="Заказать такси" onClick={onRequest} loading={loadingAction === "request-ride"} disabled={!canOrder} />
          {activeRide && <ActionButton icon={<WarningCircle weight="bold" />} label="Отменить" onClick={onCancel} loading={loadingAction === "cancel-ride"} variant="secondary" />}
        </div>
      </Surface>

      <Surface title="Статус поездки" eyebrow="Уведомления" icon={<Bell weight="bold" />}>
        {ride ? (
          <>
            <RideStatusPanel ride={ride} meta={activeMeta} />
            <RideSteps status={ride.status} />
          </>
        ) : (
          <div className="emptyState">
            <Clock weight="bold" />
            <p>Активных заказов нет.</p>
          </div>
        )}
      </Surface>
    </div>
  );
}

function DriverView({
  profile,
  waitingRides,
  currentRide,
  currentMeta,
  rideMetas,
  balance,
  loadingAction,
  onStartShift,
  onRefresh,
  onAccept,
  onArrived,
  onStartTrip,
  onComplete
}: {
  profile: DriverProfile | null;
  waitingRides: Ride[];
  currentRide: Ride | null;
  currentMeta: RideMeta | null;
  rideMetas: Record<string, RideMeta>;
  balance: number;
  loadingAction: string;
  onStartShift: () => void;
  onRefresh: () => void;
  onAccept: (ride: Ride) => void;
  onArrived: () => void;
  onStartTrip: () => void;
  onComplete: () => void;
}) {
  return (
    <div className="dashboardGrid">
      <Surface title="Смена таксиста" eyebrow="Таксист" icon={<SteeringWheel weight="bold" />}>
        <div className="routePreview">
          <StatusBadge tone={profile?.status === "online" ? "success" : "warning"}>{profile?.status === "online" ? "На линии" : "Не на линии"}</StatusBadge>
          <strong>{profile ? "Профиль таксиста готов" : "Профиль создастся автоматически"}</strong>
          <span>Заказы ниже показываются без ручного ввода ID.</span>
        </div>
        <div className="actions">
          <ActionButton icon={<CarProfile weight="bold" />} label="Начать смену" onClick={onStartShift} loading={loadingAction === "start-shift"} />
          <ActionButton icon={<Clock weight="bold" />} label="Обновить" onClick={onRefresh} loading={loadingAction === "refresh-driver"} variant="secondary" />
        </div>
      </Surface>

      <Surface title="Баланс" eyebrow="Оплата" icon={<Receipt weight="bold" />}>
        <div className="balancePanel">
          <span>Демо-баланс таксиста</span>
          <strong>{formatMoney(balance)}</strong>
          <p>При оплате картой сумма приходит во время поездки и показывается уведомлением.</p>
        </div>
      </Surface>

      <Surface title="Ожидающие заказы" eyebrow="Очередь" icon={<MapPinArea weight="bold" />} wide>
        {waitingRides.length === 0 ? (
          <div className="emptyState">
            <Clock weight="bold" />
            <p>Свободных заказов пока нет.</p>
          </div>
        ) : (
          <div className="orderList">
            {waitingRides.map((ride) => (
              <OrderCard key={ride.id} ride={ride} meta={rideMetas[ride.id]} loading={loadingAction === `accept-${ride.id}`} onAccept={() => onAccept(ride)} />
            ))}
          </div>
        )}
      </Surface>

      <Surface title="Текущая поездка" eyebrow="Маршрут" icon={<NavigationArrow weight="bold" />}>
        {currentRide ? (
          <>
            <RideStatusPanel ride={currentRide} meta={currentMeta} />
            <div className="driverActions">
              <ActionButton icon={<MapPinArea weight="bold" />} label="Я приехал" onClick={onArrived} loading={loadingAction === "ride-driver_arrived"} disabled={currentRide.status !== "driver_assigned"} />
              <ActionButton icon={<NavigationArrow weight="bold" />} label="Начать поездку" onClick={onStartTrip} loading={loadingAction === "ride-in_progress"} disabled={!["driver_assigned", "driver_arrived"].includes(currentRide.status)} />
              <ActionButton icon={<CheckCircle weight="bold" />} label="Завершить заказ" onClick={onComplete} loading={loadingAction === "ride-completed"} disabled={currentRide.status !== "in_progress"} />
            </div>
          </>
        ) : (
          <div className="emptyState">
            <CarProfile weight="bold" />
            <p>Примите заказ из очереди.</p>
          </div>
        )}
      </Surface>
    </div>
  );
}

function PlaceSelect({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="field">
      <span>{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        {PLACES.map((place) => <option key={place.id} value={place.id}>{place.name}</option>)}
      </select>
    </label>
  );
}

function PaymentOption({ active, title, text, icon, onClick }: { active: boolean; title: string; text: string; icon: ReactNode; onClick: () => void }) {
  return (
    <button className={active ? "paymentOption active" : "paymentOption"} type="button" onClick={onClick}>
      {icon}
      <span>
        <strong>{title}</strong>
        <small>{text}</small>
      </span>
      <CheckCircle weight={active ? "fill" : "regular"} />
    </button>
  );
}

function OrderCard({ ride, meta, loading, onAccept }: { ride: Ride; meta?: RideMeta; loading: boolean; onAccept: () => void }) {
  return (
    <article className="orderCard">
      <div>
        <StatusBadge tone="warning">Ждёт таксиста</StatusBadge>
        <strong>{formatMoney(ride.price)}</strong>
      </div>
      <p>{meta?.pickup ?? "Точка посадки"} → {meta?.dropoff ?? "Пункт назначения"}</p>
      <div className="orderMeta">
        <span>{meta?.paymentMethod === "card" ? "Карта" : "Наличка"}</span>
        <span>{meta?.passengerName ?? "Клиент"}</span>
      </div>
      <ActionButton icon={<CheckCircle weight="bold" />} label="Принять заказ" onClick={onAccept} loading={loading} />
    </article>
  );
}

function RideStatusPanel({ ride, meta }: { ride: Ride; meta?: RideMeta | null }) {
  return (
    <div className="ridePanel">
      <StatusBadge tone={statusTone(ride.status)}>{statusText[ride.status] ?? ride.status}</StatusBadge>
      <strong>{meta?.pickup ?? "Точка посадки"} → {meta?.dropoff ?? "Пункт назначения"}</strong>
      <span>{formatMoney(ride.price)} · {meta?.paymentMethod === "card" ? `Карта${meta.cardLast4 ? ` • ${meta.cardLast4}` : ""}` : "Наличка"}</span>
      {meta?.paid && <span className="paidLine">Оплата картой прошла.</span>}
    </div>
  );
}

function RideSteps({ status }: { status: string }) {
  const steps = ["requested", "driver_assigned", "driver_arrived", "in_progress", "completed"];
  const index = Math.max(0, steps.indexOf(status));
  return (
    <div className="rideSteps">
      {steps.map((step, stepIndex) => (
        <span key={step} className={stepIndex <= index ? "done" : ""}>{statusText[step]}</span>
      ))}
    </div>
  );
}

function ToastStack({ toasts }: { toasts: Toast[] }) {
  return (
    <div className="toastStack" aria-live="polite">
      {toasts.map((toast) => (
        <article key={toast.id} className={`toast ${toast.tone}`}>
          <Bell weight="bold" />
          <div>
            <strong>{toast.title}</strong>
            <span>{toast.message}</span>
          </div>
        </article>
      ))}
    </div>
  );
}

function statusMessage(ride: Ride, meta?: RideMeta) {
  switch (ride.status) {
    case "driver_assigned":
      return "Таксист принял заказ и едет к клиенту.";
    case "driver_arrived":
      return "Таксист уже на месте посадки.";
    case "in_progress":
      return meta?.paymentMethod === "card" ? "Поездка началась, демо-оплата картой проводится." : "Поездка началась.";
    case "completed":
      return "Поездка завершена.";
    case "cancelled":
      return "Заказ отменён.";
    default:
      return "Ждём свободного таксиста.";
  }
}
