package router

import (
	"net/http"

	"github.com/jetkzu/jetkzu/gateway/internal/handlers"
	"github.com/jetkzu/jetkzu/gateway/internal/middleware"
	"github.com/jetkzu/jetkzu/pkg/jwt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func New(h *handlers.Handlers, jm *jwt.Manager, log *zap.Logger) http.Handler {
	mux := http.NewServeMux()

	// Health and metrics
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("GET /metrics", promhttp.Handler())

	// Public endpoints
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("POST /api/auth/validate", h.ValidateSession)
	mux.HandleFunc("POST /api/users/verify-email", h.VerifyEmail)
	mux.HandleFunc("POST /api/users/reset-password", h.ResetPassword)

	mux.HandleFunc("POST /api/rides/estimate", h.EstimateRide)
	mux.HandleFunc("GET /api/drivers/nearest", h.FindNearestDrivers)
	mux.HandleFunc("GET /api/drivers", h.ListDrivers)
	mux.HandleFunc("GET /api/drivers/available", h.ListAvailableDrivers)
	mux.HandleFunc("GET /api/drivers/{id}/location", h.GetDriverLocation)
	mux.HandleFunc("POST /api/drivers/assign", h.AssignDriver)
	mux.HandleFunc("GET /api/rides/active", h.ListActiveRides)
	mux.HandleFunc("GET /api/drivers/{driver_id}/rides", h.ListDriverRides)
	mux.HandleFunc("GET /api/rides/{id}/history", h.GetRideHistory)
	mux.HandleFunc("GET /api/payments/{id}", h.GetPayment)
	mux.HandleFunc("GET /api/payments/{id}/receipt", h.GetPaymentReceipt)
	mux.HandleFunc("POST /api/payments/validate-method", h.ValidatePaymentMethod)
	mux.HandleFunc("GET /api/payments/failed", h.ListFailedPayments)

	auth := middleware.Auth(jm)

	// Protected endpoints
	mux.Handle("GET /api/users/me", auth(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/users/me", auth(http.HandlerFunc(h.UpdateMe)))
	mux.Handle("GET /api/users", auth(http.HandlerFunc(h.ListUsers)))
	mux.Handle("GET /api/users/by-email", auth(http.HandlerFunc(h.GetUserByEmail)))
	mux.Handle("POST /api/users/change-password", auth(http.HandlerFunc(h.ChangePassword)))
	mux.Handle("POST /api/users/resend-verification", auth(http.HandlerFunc(h.ResendVerification)))
	mux.Handle("POST /api/users/{id}/deactivate", auth(http.HandlerFunc(h.DeactivateUser)))
	mux.Handle("PATCH /api/users/{id}/role", auth(http.HandlerFunc(h.UpdateUserRole)))

	mux.Handle("POST /api/drivers/register", auth(http.HandlerFunc(h.RegisterDriver)))
	mux.Handle("POST /api/drivers/vehicle", auth(http.HandlerFunc(h.AddVehicle)))
	mux.Handle("PUT /api/drivers/vehicle/{id}", auth(http.HandlerFunc(h.UpdateVehicle)))
	mux.Handle("DELETE /api/drivers/vehicle/{id}", auth(http.HandlerFunc(h.DeleteVehicle)))
	mux.Handle("PATCH /api/drivers/status", auth(http.HandlerFunc(h.UpdateDriverStatus)))
	mux.Handle("PATCH /api/drivers/location", auth(http.HandlerFunc(h.UpdateDriverLocation)))
	mux.Handle("GET /api/drivers/{id}/status-history", auth(http.HandlerFunc(h.GetDriverStatusHistory)))
	mux.Handle("POST /api/drivers/{id}/rating", auth(http.HandlerFunc(h.SetDriverRating)))

	mux.Handle("POST /api/rides", auth(http.HandlerFunc(h.CreateRide)))
	mux.Handle("POST /api/rides/schedule", auth(http.HandlerFunc(h.ScheduleRide)))
	mux.Handle("GET /api/rides/my", auth(http.HandlerFunc(h.MyRides)))
	mux.Handle("GET /api/rides/{id}", auth(http.HandlerFunc(h.GetRide)))
	mux.Handle("POST /api/rides/{id}/accept", auth(http.HandlerFunc(h.AcceptRide)))
	mux.Handle("POST /api/rides/{id}/reject", auth(http.HandlerFunc(h.RejectRide)))
	mux.Handle("PATCH /api/rides/{id}/status", auth(http.HandlerFunc(h.UpdateRideStatus)))
	mux.Handle("POST /api/rides/{id}/cancel", auth(http.HandlerFunc(h.CancelRide)))
	mux.Handle("POST /api/rides/{id}/complete", auth(http.HandlerFunc(h.CompleteRide)))
	mux.Handle("POST /api/rides/{id}/rate", auth(http.HandlerFunc(h.RateRide)))

	mux.Handle("POST /api/payments", auth(http.HandlerFunc(h.CreatePayment)))
	mux.Handle("GET /api/rides/{ride_id}/payment", auth(http.HandlerFunc(h.GetPaymentByRide)))
	mux.Handle("GET /api/payments/my", auth(http.HandlerFunc(h.MyPayments)))
	mux.Handle("POST /api/payments/process", auth(http.HandlerFunc(h.ProcessPayment)))
	mux.Handle("POST /api/payments/refund-request", auth(http.HandlerFunc(h.CreateRefundRequest)))
	mux.Handle("POST /api/payments/refund", auth(http.HandlerFunc(h.RefundPayment)))

	mux.Handle("POST /api/notifications/email", auth(http.HandlerFunc(h.SendEmail)))
	mux.Handle("POST /api/notifications/ride-receipt", auth(http.HandlerFunc(h.SendRideReceipt)))
	mux.Handle("GET /api/notifications/my", auth(http.HandlerFunc(h.MyNotifications)))
	mux.Handle("GET /api/notifications/unread", auth(http.HandlerFunc(h.UnreadNotifications)))
	mux.Handle("GET /api/notifications/unread/count", auth(http.HandlerFunc(h.CountUnreadNotifications)))
	mux.Handle("PATCH /api/notifications/{id}/read", auth(http.HandlerFunc(h.MarkNotificationRead)))
	mux.Handle("PATCH /api/notifications/read-all", auth(http.HandlerFunc(h.MarkAllNotificationsRead)))
	mux.Handle("POST /api/notifications/{id}/resend", auth(http.HandlerFunc(h.ResendNotification)))
	mux.Handle("DELETE /api/notifications/{id}", auth(http.HandlerFunc(h.DeleteNotification)))

	return middleware.CORS(middleware.CorrelationID(middleware.Logging(log)(mux)))
}
