package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jetkzu/jetkzu/gateway/internal/clients"
	"github.com/jetkzu/jetkzu/gateway/internal/middleware"
	driverv1 "github.com/jetkzu/jetkzu/gen/go/driver/v1"
	notifv1 "github.com/jetkzu/jetkzu/gen/go/notification/v1"
	paymentv1 "github.com/jetkzu/jetkzu/gen/go/payment/v1"
	ridev1 "github.com/jetkzu/jetkzu/gen/go/ride/v1"
	userv1 "github.com/jetkzu/jetkzu/gen/go/user/v1"
	"github.com/jetkzu/jetkzu/pkg/httpx"
	"github.com/jetkzu/jetkzu/pkg/logger"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handlers struct {
	C       *clients.Clients
	Timeout time.Duration
}

func New(c *clients.Clients, timeout time.Duration) *Handlers {
	return &Handlers{C: c, Timeout: timeout}
}

func (h *Handlers) ctx(r *http.Request) (context.Context, context.CancelFunc) {
	ctx := middleware.GRPCContext(r.Context())
	return context.WithTimeout(ctx, h.Timeout)
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func writeGRPCError(w http.ResponseWriter, r *http.Request, err error) {
	cid := logger.CorrelationIDFrom(r.Context())
	st, ok := status.FromError(err)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error(), cid)
		return
	}
	code := http.StatusInternalServerError
	switch st.Code().String() {
	case "NotFound":
		code = http.StatusNotFound
	case "AlreadyExists":
		code = http.StatusConflict
	case "InvalidArgument":
		code = http.StatusBadRequest
	case "Unauthenticated":
		code = http.StatusUnauthorized
	case "PermissionDenied":
		code = http.StatusForbidden
	case "FailedPrecondition":
		code = http.StatusConflict
	}
	httpx.WriteError(w, code, st.Message(), cid)
}

// ---------- Auth / Users ----------

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var in registerReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.RegisterUser(ctx, &userv1.RegisterUserRequest{
		Email: in.Email, Password: in.Password, FullName: in.FullName, Phone: in.Phone, Role: in.Role,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var in loginReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.LoginUser(ctx, &userv1.LoginUserRequest{Email: in.Email, Password: in.Password})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID string `json:"user_id"`
	}
	_ = decode(r, &in)
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.LogoutUser(ctx, &userv1.LogoutUserRequest{UserId: in.UserID})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ValidateSession(w http.ResponseWriter, r *http.Request) {
	var in struct {
		AccessToken string `json:"access_token"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.ValidateSession(ctx, &userv1.ValidateSessionRequest{AccessToken: in.AccessToken})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "no user", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.GetUserProfile(ctx, &userv1.GetUserProfileRequest{UserId: uid})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.GetUserByEmail(ctx, &userv1.GetUserByEmailRequest{Email: r.URL.Query().Get("email")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.ListUsers(ctx, &userv1.ListUsersRequest{
		Limit: int32(limit), Offset: int32(offset), Role: q.Get("role"),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type updateMeReq struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

func (h *Handlers) UpdateMe(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "no user", "")
		return
	}
	var in updateMeReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.UpdateUserProfile(ctx, &userv1.UpdateUserProfileRequest{
		UserId: uid, FullName: in.FullName, Phone: in.Phone,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.ChangePassword(ctx, &userv1.ChangePasswordRequest{
		UserId: middleware.UserID(r.Context()), OldPassword: in.OldPassword, NewPassword: in.NewPassword,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email       string `json:"email"`
		NewPassword string `json:"new_password"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.ResetPassword(ctx, &userv1.ResetPasswordRequest{Email: in.Email, NewPassword: in.NewPassword})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type verifyReq struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

func (h *Handlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var in verifyReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.VerifyUserEmail(ctx, &userv1.VerifyUserEmailRequest{UserId: in.UserID, Token: in.Token})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ResendVerification(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.ResendVerification(ctx, &userv1.ResendVerificationRequest{UserId: middleware.UserID(r.Context())})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) DeactivateUser(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.DeactivateUser(ctx, &userv1.DeactivateUserRequest{UserId: r.PathValue("id")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Role string `json:"role"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.User.UpdateUserRole(ctx, &userv1.UpdateUserRoleRequest{UserId: r.PathValue("id"), Role: in.Role})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// ---------- Drivers ----------

type registerDriverReq struct {
	UserID        string `json:"user_id"`
	LicenseNumber string `json:"license_number"`
}

func (h *Handlers) RegisterDriver(w http.ResponseWriter, r *http.Request) {
	var in registerDriverReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	if in.UserID == "" {
		in.UserID = middleware.UserID(r.Context())
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.RegisterDriver(ctx, &driverv1.RegisterDriverRequest{
		UserId: in.UserID, LicenseNumber: in.LicenseNumber,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

type addVehicleReq struct {
	DriverID    string `json:"driver_id"`
	PlateNumber string `json:"plate_number"`
	Make        string `json:"make"`
	Model       string `json:"model"`
	Year        int32  `json:"year"`
	Color       string `json:"color"`
}

func (h *Handlers) AddVehicle(w http.ResponseWriter, r *http.Request) {
	var in addVehicleReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.AddVehicle(ctx, &driverv1.AddVehicleRequest{
		DriverId: in.DriverID, PlateNumber: in.PlateNumber, Make: in.Make,
		Model: in.Model, Year: in.Year, Color: in.Color,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handlers) UpdateVehicle(w http.ResponseWriter, r *http.Request) {
	var in addVehicleReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.UpdateVehicle(ctx, &driverv1.UpdateVehicleRequest{
		VehicleId: r.PathValue("id"), PlateNumber: in.PlateNumber, Make: in.Make,
		Model: in.Model, Year: in.Year, Color: in.Color,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) DeleteVehicle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.DeleteVehicle(ctx, &driverv1.DeleteVehicleRequest{VehicleId: r.PathValue("id")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type driverStatusReq struct {
	DriverID string `json:"driver_id"`
	Status   string `json:"status"`
}

func (h *Handlers) UpdateDriverStatus(w http.ResponseWriter, r *http.Request) {
	var in driverStatusReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.UpdateDriverStatus(ctx, &driverv1.UpdateDriverStatusRequest{
		DriverId: in.DriverID, Status: in.Status,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type driverLocReq struct {
	DriverID  string  `json:"driver_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (h *Handlers) UpdateDriverLocation(w http.ResponseWriter, r *http.Request) {
	var in driverLocReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.UpdateDriverLocation(ctx, &driverv1.UpdateDriverLocationRequest{
		DriverId: in.DriverID, Latitude: in.Latitude, Longitude: in.Longitude,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetDriverLocation(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.GetDriverLocation(ctx, &driverv1.GetDriverLocationRequest{DriverId: r.PathValue("id")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) FindNearestDrivers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, _ := parseFloat(q.Get("lat"))
	lng, _ := parseFloat(q.Get("lng"))
	radius, _ := parseFloat(q.Get("radius_km"))
	limit, _ := parseInt(q.Get("limit"))
	if radius == 0 {
		radius = 5
	}
	if limit == 0 {
		limit = 10
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.FindNearestDrivers(ctx, &driverv1.FindNearestDriversRequest{
		Latitude: lat, Longitude: lng, RadiusKm: radius, Limit: int32(limit),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ListDrivers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.ListDrivers(ctx, &driverv1.ListDriversRequest{
		Limit: int32(limit), Offset: int32(offset), Status: q.Get("status"),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ListAvailableDrivers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.ListAvailableDrivers(ctx, &driverv1.ListAvailableDriversRequest{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type assignReq struct {
	RideID          string  `json:"ride_id"`
	PickupLatitude  float64 `json:"pickup_latitude"`
	PickupLongitude float64 `json:"pickup_longitude"`
}

func (h *Handlers) AssignDriver(w http.ResponseWriter, r *http.Request) {
	var in assignReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.AssignDriverToRide(ctx, &driverv1.AssignDriverToRideRequest{
		RideId: in.RideID, PickupLatitude: in.PickupLatitude, PickupLongitude: in.PickupLongitude,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetDriverStatusHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := parseInt(r.URL.Query().Get("limit"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.GetDriverStatusHistory(ctx, &driverv1.GetDriverStatusHistoryRequest{
		DriverId: r.PathValue("id"), Limit: int32(limit),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) SetDriverRating(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Rating float64 `json:"rating"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Driver.SetDriverRating(ctx, &driverv1.SetDriverRatingRequest{DriverId: r.PathValue("id"), Rating: in.Rating})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// ---------- Rides ----------

type createRideReq struct {
	PassengerID string  `json:"passenger_id"`
	PickupLat   float64 `json:"pickup_lat"`
	PickupLng   float64 `json:"pickup_lng"`
	DropoffLat  float64 `json:"dropoff_lat"`
	DropoffLng  float64 `json:"dropoff_lng"`
}

func (h *Handlers) CreateRide(w http.ResponseWriter, r *http.Request) {
	var in createRideReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	if in.PassengerID == "" {
		in.PassengerID = middleware.UserID(r.Context())
	}
	if in.PassengerID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "passenger_id required", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.CreateRide(ctx, &ridev1.CreateRideRequest{
		PassengerId: in.PassengerID, PickupLat: in.PickupLat, PickupLng: in.PickupLng,
		DropoffLat: in.DropoffLat, DropoffLng: in.DropoffLng,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handlers) ScheduleRide(w http.ResponseWriter, r *http.Request) {
	var in struct {
		createRideReq
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	if in.PassengerID == "" {
		in.PassengerID = middleware.UserID(r.Context())
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.ScheduleRide(ctx, &ridev1.ScheduleRideRequest{
		PassengerId: in.PassengerID, PickupLat: in.PickupLat, PickupLng: in.PickupLng,
		DropoffLat: in.DropoffLat, DropoffLng: in.DropoffLng, ScheduledAt: timestamppb.New(in.ScheduledAt),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

type estimateReq struct {
	PickupLat  float64 `json:"pickup_lat"`
	PickupLng  float64 `json:"pickup_lng"`
	DropoffLat float64 `json:"dropoff_lat"`
	DropoffLng float64 `json:"dropoff_lng"`
}

func (h *Handlers) EstimateRide(w http.ResponseWriter, r *http.Request) {
	var in estimateReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.EstimateRidePrice(ctx, &ridev1.EstimateRidePriceRequest{
		PickupLat: in.PickupLat, PickupLng: in.PickupLng, DropoffLat: in.DropoffLat, DropoffLng: in.DropoffLng,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ListActiveRides(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.ListActiveRides(ctx, &ridev1.ListActiveRidesRequest{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ListDriverRides(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.ListDriverRides(ctx, &ridev1.ListDriverRidesRequest{
		DriverId: r.PathValue("driver_id"), Limit: int32(limit), Offset: int32(offset),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetRide(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.GetRide(ctx, &ridev1.GetRideRequest{RideId: id})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetRideHistory(w http.ResponseWriter, r *http.Request) {
	limit, _ := parseInt(r.URL.Query().Get("limit"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.GetRideHistory(ctx, &ridev1.GetRideHistoryRequest{RideId: r.PathValue("id"), Limit: int32(limit)})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) MyRides(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "no user", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.ListUserRides(ctx, &ridev1.ListUserRidesRequest{UserId: uid, Limit: 50})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) AcceptRide(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DriverID string `json:"driver_id"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.AcceptRide(ctx, &ridev1.AcceptRideRequest{RideId: r.PathValue("id"), DriverId: in.DriverID})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) RejectRide(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DriverID string `json:"driver_id"`
		Reason   string `json:"reason"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.RejectRide(ctx, &ridev1.RejectRideRequest{RideId: r.PathValue("id"), DriverId: in.DriverID, Reason: in.Reason})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type updateStatusReq struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (h *Handlers) UpdateRideStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in updateStatusReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.UpdateRideStatus(ctx, &ridev1.UpdateRideStatusRequest{
		RideId: id, Status: in.Status, Reason: in.Reason,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type cancelReq struct {
	Reason string `json:"reason"`
}

func (h *Handlers) CancelRide(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in cancelReq
	_ = decode(r, &in)
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.CancelRide(ctx, &ridev1.CancelRideRequest{RideId: id, Reason: in.Reason})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) CompleteRide(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.CompleteRide(ctx, &ridev1.CompleteRideRequest{RideId: id})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) RateRide(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID  string `json:"user_id"`
		Rating  int32  `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	if in.UserID == "" {
		in.UserID = middleware.UserID(r.Context())
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Ride.RateRide(ctx, &ridev1.RateRideRequest{
		RideId: r.PathValue("id"), UserId: in.UserID, Rating: in.Rating, Comment: in.Comment,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// ---------- Payments ----------

type createPaymentReq struct {
	RideID string  `json:"ride_id"`
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
	Method string  `json:"method"`
}

func (h *Handlers) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var in createPaymentReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	if in.UserID == "" {
		in.UserID = middleware.UserID(r.Context())
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
		RideId: in.RideID, UserId: in.UserID, Amount: in.Amount, Method: in.Method,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handlers) GetPaymentByRide(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("ride_id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "ride_id required", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.GetPaymentByRide(ctx, &paymentv1.GetPaymentByRideRequest{RideId: id})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetPayment(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.GetPayment(ctx, &paymentv1.GetPaymentRequest{PaymentId: r.PathValue("id")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) MyPayments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.ListUserPayments(ctx, &paymentv1.ListUserPaymentsRequest{
		UserId: middleware.UserID(r.Context()), Limit: int32(limit), Offset: int32(offset),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ListFailedPayments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.ListFailedPayments(ctx, &paymentv1.ListFailedPaymentsRequest{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) GetPaymentReceipt(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.GetPaymentReceipt(ctx, &paymentv1.GetPaymentReceiptRequest{PaymentId: r.PathValue("id")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ValidatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Method string `json:"method"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.ValidatePaymentMethod(ctx, &paymentv1.ValidatePaymentMethodRequest{Method: in.Method})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type processPaymentReq struct {
	PaymentID string `json:"payment_id"`
}

func (h *Handlers) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	var in processPaymentReq
	if err := decode(r, &in); err != nil || in.PaymentID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "payment_id required", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{PaymentId: in.PaymentID})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

type refundPaymentReq struct {
	PaymentID string `json:"payment_id"`
	Reason    string `json:"reason"`
}

func (h *Handlers) RefundPayment(w http.ResponseWriter, r *http.Request) {
	var in refundPaymentReq
	if err := decode(r, &in); err != nil || in.PaymentID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "payment_id required", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.RefundPayment(ctx, &paymentv1.RefundPaymentRequest{
		PaymentId: in.PaymentID, Reason: in.Reason,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) CreateRefundRequest(w http.ResponseWriter, r *http.Request) {
	var in refundPaymentReq
	if err := decode(r, &in); err != nil || in.PaymentID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "payment_id required", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Payment.CreateRefundRequest(ctx, &paymentv1.CreateRefundRequestRequest{
		PaymentId: in.PaymentID, Reason: in.Reason,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// ---------- Notifications ----------

type sendEmailReq struct {
	UserID  string `json:"user_id"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (h *Handlers) SendEmail(w http.ResponseWriter, r *http.Request) {
	var in sendEmailReq
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	if in.UserID == "" {
		in.UserID = middleware.UserID(r.Context())
	}
	if in.UserID == "" || in.To == "" {
		httpx.WriteError(w, http.StatusBadRequest, "user_id and to are required", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.SendEmailNotification(ctx, &notifv1.SendEmailNotificationRequest{
		UserId: in.UserID, To: in.To, Subject: in.Subject, Body: in.Body,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handlers) SendRideReceipt(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID    string  `json:"user_id"`
		To        string  `json:"to"`
		RideID    string  `json:"ride_id"`
		PaymentID string  `json:"payment_id"`
		Amount    float64 `json:"amount"`
	}
	if err := decode(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body", "")
		return
	}
	if in.UserID == "" {
		in.UserID = middleware.UserID(r.Context())
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.SendRideReceipt(ctx, &notifv1.SendRideReceiptRequest{
		UserId: in.UserID, To: in.To, RideId: in.RideID, PaymentId: in.PaymentID, Amount: in.Amount,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handlers) MyNotifications(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "no user", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.GetNotificationHistory(ctx, &notifv1.GetNotificationHistoryRequest{
		UserId: uid, Limit: 50,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) UnreadNotifications(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := parseInt(q.Get("limit"))
	offset, _ := parseInt(q.Get("offset"))
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.ListUnreadNotifications(ctx, &notifv1.ListUnreadNotificationsRequest{
		UserId: middleware.UserID(r.Context()), Limit: int32(limit), Offset: int32(offset),
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) CountUnreadNotifications(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.CountUnreadNotifications(ctx, &notifv1.CountUnreadNotificationsRequest{UserId: middleware.UserID(r.Context())})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "id required", "")
		return
	}
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.MarkNotificationAsRead(ctx, &notifv1.MarkNotificationAsReadRequest{
		NotificationId: id,
	})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) MarkAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.MarkAllNotificationsAsRead(ctx, &notifv1.MarkAllNotificationsAsReadRequest{UserId: middleware.UserID(r.Context())})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) ResendNotification(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.ResendNotification(ctx, &notifv1.ResendNotificationRequest{NotificationId: r.PathValue("id")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h *Handlers) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := h.ctx(r)
	defer cancel()
	res, err := h.C.Notification.DeleteNotification(ctx, &notifv1.DeleteNotificationRequest{NotificationId: r.PathValue("id")})
	if err != nil {
		writeGRPCError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, res)
}

// helpers

func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	var f float64
	_, err := fmt.Sscan(s, &f)
	return f, err
}
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty")
	}
	var n int
	_, err := fmt.Sscan(s, &n)
	return n, err
}
