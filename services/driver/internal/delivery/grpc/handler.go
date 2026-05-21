package grpc

import (
	"context"

	driverv1 "github.com/jetkzu/jetkzu/gen/go/driver/v1"
	"github.com/jetkzu/jetkzu/services/driver/internal/domain"
	"github.com/jetkzu/jetkzu/services/driver/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	driverv1.UnimplementedDriverServiceServer
	uc *usecase.UseCase
}

func NewHandler(uc *usecase.UseCase) *Handler { return &Handler{uc: uc} }

func toPB(d *domain.Driver) *driverv1.Driver {
	if d == nil {
		return nil
	}
	return &driverv1.Driver{
		Id: d.ID, UserId: d.UserID, LicenseNumber: d.LicenseNumber,
		Status: d.Status, Latitude: d.Latitude, Longitude: d.Longitude,
		CreatedAt: timestamppb.New(d.CreatedAt),
	}
}

func vehiclePB(v *domain.Vehicle) *driverv1.Vehicle {
	if v == nil {
		return nil
	}
	return &driverv1.Vehicle{
		Id: v.ID, DriverId: v.DriverID, PlateNumber: v.PlateNumber,
		Make: v.Make, Model: v.Model, Year: v.Year, Color: v.Color,
	}
}

func (h *Handler) RegisterDriver(ctx context.Context, req *driverv1.RegisterDriverRequest) (*driverv1.RegisterDriverResponse, error) {
	d, err := h.uc.Register(ctx, req.UserId, req.LicenseNumber)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &driverv1.RegisterDriverResponse{Driver: toPB(d)}, nil
}

func (h *Handler) AddVehicle(ctx context.Context, req *driverv1.AddVehicleRequest) (*driverv1.AddVehicleResponse, error) {
	v, err := h.uc.AddVehicle(ctx, &domain.Vehicle{
		DriverID: req.DriverId, PlateNumber: req.PlateNumber, Make: req.Make,
		Model: req.Model, Year: req.Year, Color: req.Color,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &driverv1.AddVehicleResponse{Vehicle: vehiclePB(v)}, nil
}

func (h *Handler) UpdateVehicle(ctx context.Context, req *driverv1.UpdateVehicleRequest) (*driverv1.UpdateVehicleResponse, error) {
	v, err := h.uc.UpdateVehicle(ctx, &domain.Vehicle{
		ID: req.VehicleId, PlateNumber: req.PlateNumber, Make: req.Make,
		Model: req.Model, Year: req.Year, Color: req.Color,
	})
	if err != nil {
		if err == domain.ErrDriverNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &driverv1.UpdateVehicleResponse{Vehicle: vehiclePB(v)}, nil
}

func (h *Handler) DeleteVehicle(ctx context.Context, req *driverv1.DeleteVehicleRequest) (*driverv1.DeleteVehicleResponse, error) {
	ok, err := h.uc.DeleteVehicle(ctx, req.VehicleId)
	if err != nil {
		if err == domain.ErrDriverNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &driverv1.DeleteVehicleResponse{Deleted: ok}, nil
}

func (h *Handler) UpdateDriverStatus(ctx context.Context, req *driverv1.UpdateDriverStatusRequest) (*driverv1.UpdateDriverStatusResponse, error) {
	d, err := h.uc.UpdateStatus(ctx, req.DriverId, req.Status)
	if err != nil {
		if err == domain.ErrDriverNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &driverv1.UpdateDriverStatusResponse{Driver: toPB(d)}, nil
}

func (h *Handler) UpdateDriverLocation(ctx context.Context, req *driverv1.UpdateDriverLocationRequest) (*driverv1.UpdateDriverLocationResponse, error) {
	if err := h.uc.UpdateLocation(ctx, req.DriverId, req.Latitude, req.Longitude); err != nil {
		if err == domain.ErrDriverNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &driverv1.UpdateDriverLocationResponse{Ok: true}, nil
}

func (h *Handler) GetDriverLocation(ctx context.Context, req *driverv1.GetDriverLocationRequest) (*driverv1.GetDriverLocationResponse, error) {
	d, err := h.uc.GetLocation(ctx, req.DriverId)
	if err != nil {
		if err == domain.ErrDriverNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &driverv1.GetDriverLocationResponse{DriverId: d.ID, Latitude: d.Latitude, Longitude: d.Longitude}, nil
}

func (h *Handler) FindNearestDrivers(ctx context.Context, req *driverv1.FindNearestDriversRequest) (*driverv1.FindNearestDriversResponse, error) {
	list, err := h.uc.FindNearest(ctx, req.Latitude, req.Longitude, req.RadiusKm, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*driverv1.NearbyDriver, 0, len(list))
	for _, n := range list {
		out = append(out, &driverv1.NearbyDriver{
			DriverId: n.DriverID, Latitude: n.Latitude, Longitude: n.Longitude, DistanceKm: n.DistanceKm,
		})
	}
	return &driverv1.FindNearestDriversResponse{Drivers: out}, nil
}

func (h *Handler) ListDrivers(ctx context.Context, req *driverv1.ListDriversRequest) (*driverv1.ListDriversResponse, error) {
	drivers, err := h.uc.List(ctx, req.Status, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	out := make([]*driverv1.Driver, 0, len(drivers))
	for _, d := range drivers {
		out = append(out, toPB(d))
	}
	return &driverv1.ListDriversResponse{Drivers: out}, nil
}

func (h *Handler) ListAvailableDrivers(ctx context.Context, req *driverv1.ListAvailableDriversRequest) (*driverv1.ListAvailableDriversResponse, error) {
	drivers, err := h.uc.ListAvailable(ctx, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*driverv1.Driver, 0, len(drivers))
	for _, d := range drivers {
		out = append(out, toPB(d))
	}
	return &driverv1.ListAvailableDriversResponse{Drivers: out}, nil
}

func (h *Handler) AssignDriverToRide(ctx context.Context, req *driverv1.AssignDriverToRideRequest) (*driverv1.AssignDriverToRideResponse, error) {
	driverID, err := h.uc.AssignToRide(ctx, req.RideId, req.PickupLatitude, req.PickupLongitude)
	if err != nil {
		if err == domain.ErrNoDriverNearby {
			return &driverv1.AssignDriverToRideResponse{Assigned: false}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &driverv1.AssignDriverToRideResponse{DriverId: driverID, Assigned: true}, nil
}

func (h *Handler) GetDriver(ctx context.Context, req *driverv1.GetDriverRequest) (*driverv1.GetDriverResponse, error) {
	d, v, err := h.uc.Get(ctx, req.DriverId)
	if err != nil {
		if err == domain.ErrDriverNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &driverv1.GetDriverResponse{Driver: toPB(d), Vehicle: vehiclePB(v)}, nil
}

func (h *Handler) GetDriverStatusHistory(ctx context.Context, req *driverv1.GetDriverStatusHistoryRequest) (*driverv1.GetDriverStatusHistoryResponse, error) {
	list, err := h.uc.StatusHistory(ctx, req.DriverId, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*driverv1.DriverStatusHistory, 0, len(list))
	for _, hst := range list {
		out = append(out, &driverv1.DriverStatusHistory{
			DriverId: hst.DriverID, Status: hst.Status, ChangedAt: timestamppb.New(hst.ChangedAt),
		})
	}
	return &driverv1.GetDriverStatusHistoryResponse{History: out}, nil
}

func (h *Handler) SetDriverRating(ctx context.Context, req *driverv1.SetDriverRatingRequest) (*driverv1.SetDriverRatingResponse, error) {
	rating, err := h.uc.SetRating(ctx, req.DriverId, req.Rating)
	if err != nil {
		if err == domain.ErrDriverNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &driverv1.SetDriverRatingResponse{DriverId: req.DriverId, Rating: rating}, nil
}
