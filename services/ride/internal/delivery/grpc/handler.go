package grpc

import (
	"context"

	ridev1 "github.com/jetkzu/jetkzu/gen/go/ride/v1"
	"github.com/jetkzu/jetkzu/services/ride/internal/domain"
	"github.com/jetkzu/jetkzu/services/ride/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	ridev1.UnimplementedRideServiceServer
	uc *usecase.UseCase
}

func NewHandler(uc *usecase.UseCase) *Handler { return &Handler{uc: uc} }

func toPB(r *domain.Ride) *ridev1.Ride {
	if r == nil {
		return nil
	}
	return &ridev1.Ride{
		Id: r.ID, PassengerId: r.PassengerID, DriverId: r.DriverID,
		PickupLat: r.PickupLat, PickupLng: r.PickupLng,
		DropoffLat: r.DropoffLat, DropoffLng: r.DropoffLng,
		Status: r.Status, Price: r.Price,
		CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}

func (h *Handler) CreateRide(ctx context.Context, req *ridev1.CreateRideRequest) (*ridev1.CreateRideResponse, error) {
	r, err := h.uc.Create(ctx, usecase.CreateRideInput{
		PassengerID: req.PassengerId, PickupLat: req.PickupLat, PickupLng: req.PickupLng,
		DropoffLat: req.DropoffLat, DropoffLng: req.DropoffLng,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.CreateRideResponse{Ride: toPB(r)}, nil
}

func (h *Handler) EstimateRidePrice(ctx context.Context, req *ridev1.EstimateRidePriceRequest) (*ridev1.EstimateRidePriceResponse, error) {
	price, dist := h.uc.EstimatePrice(ctx, usecase.CreateRideInput{
		PickupLat: req.PickupLat, PickupLng: req.PickupLng, DropoffLat: req.DropoffLat, DropoffLng: req.DropoffLng,
	})
	return &ridev1.EstimateRidePriceResponse{Price: price, DistanceKm: dist}, nil
}

func (h *Handler) GetRide(ctx context.Context, req *ridev1.GetRideRequest) (*ridev1.GetRideResponse, error) {
	r, err := h.uc.Get(ctx, req.RideId)
	if err != nil {
		if err == domain.ErrRideNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.GetRideResponse{Ride: toPB(r)}, nil
}

func (h *Handler) ListUserRides(ctx context.Context, req *ridev1.ListUserRidesRequest) (*ridev1.ListUserRidesResponse, error) {
	rides, err := h.uc.ListByUser(ctx, req.UserId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*ridev1.Ride, 0, len(rides))
	for _, r := range rides {
		out = append(out, toPB(r))
	}
	return &ridev1.ListUserRidesResponse{Rides: out}, nil
}

func (h *Handler) ListActiveRides(ctx context.Context, req *ridev1.ListActiveRidesRequest) (*ridev1.ListActiveRidesResponse, error) {
	rides, err := h.uc.ListActive(ctx, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*ridev1.Ride, 0, len(rides))
	for _, r := range rides {
		out = append(out, toPB(r))
	}
	return &ridev1.ListActiveRidesResponse{Rides: out}, nil
}

func (h *Handler) ListDriverRides(ctx context.Context, req *ridev1.ListDriverRidesRequest) (*ridev1.ListDriverRidesResponse, error) {
	rides, err := h.uc.ListByDriver(ctx, req.DriverId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*ridev1.Ride, 0, len(rides))
	for _, r := range rides {
		out = append(out, toPB(r))
	}
	return &ridev1.ListDriverRidesResponse{Rides: out}, nil
}

func (h *Handler) GetRideHistory(ctx context.Context, req *ridev1.GetRideHistoryRequest) (*ridev1.GetRideHistoryResponse, error) {
	list, err := h.uc.History(ctx, req.RideId, int(req.Limit))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*ridev1.RideStatusHistory, 0, len(list))
	for _, hst := range list {
		out = append(out, &ridev1.RideStatusHistory{
			RideId: hst.RideID, Status: hst.Status, Reason: hst.Reason, ChangedAt: timestamppb.New(hst.ChangedAt),
		})
	}
	return &ridev1.GetRideHistoryResponse{History: out}, nil
}

func (h *Handler) ScheduleRide(ctx context.Context, req *ridev1.ScheduleRideRequest) (*ridev1.ScheduleRideResponse, error) {
	r, err := h.uc.Schedule(ctx, usecase.CreateRideInput{
		PassengerID: req.PassengerId, PickupLat: req.PickupLat, PickupLng: req.PickupLng,
		DropoffLat: req.DropoffLat, DropoffLng: req.DropoffLng,
	}, req.ScheduledAt.AsTime())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.ScheduleRideResponse{Ride: toPB(r)}, nil
}

func (h *Handler) AcceptRide(ctx context.Context, req *ridev1.AcceptRideRequest) (*ridev1.AcceptRideResponse, error) {
	r, err := h.uc.Accept(ctx, req.RideId, req.DriverId)
	if err != nil {
		if err == domain.ErrRideNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if err == domain.ErrInvalidTransition {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.AcceptRideResponse{Ride: toPB(r)}, nil
}

func (h *Handler) RejectRide(ctx context.Context, req *ridev1.RejectRideRequest) (*ridev1.RejectRideResponse, error) {
	ok, err := h.uc.Reject(ctx, req.RideId, req.DriverId, req.Reason)
	if err != nil {
		if err == domain.ErrRideNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.RejectRideResponse{Rejected: ok}, nil
}

func (h *Handler) UpdateRideStatus(ctx context.Context, req *ridev1.UpdateRideStatusRequest) (*ridev1.UpdateRideStatusResponse, error) {
	r, err := h.uc.UpdateStatus(ctx, req.RideId, req.Status, req.Reason)
	if err != nil {
		switch err {
		case domain.ErrRideNotFound:
			return nil, status.Error(codes.NotFound, err.Error())
		case domain.ErrInvalidTransition:
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.UpdateRideStatusResponse{Ride: toPB(r)}, nil
}

func (h *Handler) CancelRide(ctx context.Context, req *ridev1.CancelRideRequest) (*ridev1.CancelRideResponse, error) {
	r, err := h.uc.Cancel(ctx, req.RideId, req.Reason)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.CancelRideResponse{Ride: toPB(r)}, nil
}

func (h *Handler) CompleteRide(ctx context.Context, req *ridev1.CompleteRideRequest) (*ridev1.CompleteRideResponse, error) {
	r, err := h.uc.Complete(ctx, req.RideId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.CompleteRideResponse{Ride: toPB(r)}, nil
}

func (h *Handler) RateRide(ctx context.Context, req *ridev1.RateRideRequest) (*ridev1.RateRideResponse, error) {
	rating, err := h.uc.Rate(ctx, req.RideId, req.UserId, req.Rating, req.Comment)
	if err != nil {
		if err == domain.ErrRideNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &ridev1.RateRideResponse{RideId: req.RideId, Rating: rating}, nil
}
