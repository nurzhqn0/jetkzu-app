package grpc

import (
	"context"

	paymentv1 "github.com/jetkzu/jetkzu/gen/go/payment/v1"
	"github.com/jetkzu/jetkzu/services/payment/internal/domain"
	"github.com/jetkzu/jetkzu/services/payment/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	paymentv1.UnimplementedPaymentServiceServer
	uc *usecase.UseCase
}

func NewHandler(uc *usecase.UseCase) *Handler { return &Handler{uc: uc} }

func toPB(p *domain.Payment) *paymentv1.Payment {
	if p == nil {
		return nil
	}
	return &paymentv1.Payment{
		Id: p.ID, RideId: p.RideID, UserId: p.UserID, Amount: p.Amount, Currency: p.Currency,
		Status: p.Status, Method: p.Method,
		CreatedAt: timestamppb.New(p.CreatedAt), UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}

func (h *Handler) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	p, err := h.uc.Create(ctx, usecase.CreateInput{
		RideID: req.RideId, UserID: req.UserId, Amount: req.Amount, Method: req.Method,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.CreatePaymentResponse{Payment: toPB(p)}, nil
}

func (h *Handler) GetPaymentByRide(ctx context.Context, req *paymentv1.GetPaymentByRideRequest) (*paymentv1.GetPaymentByRideResponse, error) {
	p, err := h.uc.GetByRide(ctx, req.RideId)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.GetPaymentByRideResponse{Payment: toPB(p)}, nil
}

func (h *Handler) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	p, err := h.uc.Get(ctx, req.PaymentId)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.GetPaymentResponse{Payment: toPB(p)}, nil
}

func (h *Handler) ListUserPayments(ctx context.Context, req *paymentv1.ListUserPaymentsRequest) (*paymentv1.ListUserPaymentsResponse, error) {
	payments, err := h.uc.ListByUser(ctx, req.UserId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*paymentv1.Payment, 0, len(payments))
	for _, p := range payments {
		out = append(out, toPB(p))
	}
	return &paymentv1.ListUserPaymentsResponse{Payments: out}, nil
}

func (h *Handler) ListFailedPayments(ctx context.Context, req *paymentv1.ListFailedPaymentsRequest) (*paymentv1.ListFailedPaymentsResponse, error) {
	payments, err := h.uc.ListFailed(ctx, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*paymentv1.Payment, 0, len(payments))
	for _, p := range payments {
		out = append(out, toPB(p))
	}
	return &paymentv1.ListFailedPaymentsResponse{Payments: out}, nil
}

func (h *Handler) GetPaymentReceipt(ctx context.Context, req *paymentv1.GetPaymentReceiptRequest) (*paymentv1.GetPaymentReceiptResponse, error) {
	p, summary, err := h.uc.Receipt(ctx, req.PaymentId)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.GetPaymentReceiptResponse{ReceiptId: "receipt-" + p.ID, Payment: toPB(p), Summary: summary}, nil
}

func (h *Handler) ValidatePaymentMethod(ctx context.Context, req *paymentv1.ValidatePaymentMethodRequest) (*paymentv1.ValidatePaymentMethodResponse, error) {
	ok, msg := h.uc.ValidateMethod(req.Method)
	return &paymentv1.ValidatePaymentMethodResponse{Valid: ok, Message: msg}, nil
}

func (h *Handler) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	p, err := h.uc.Process(ctx, req.PaymentId)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if err == domain.ErrInvalidTransition {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.ProcessPaymentResponse{Payment: toPB(p)}, nil
}

func (h *Handler) CreateRefundRequest(ctx context.Context, req *paymentv1.CreateRefundRequestRequest) (*paymentv1.CreateRefundRequestResponse, error) {
	p, refundID, err := h.uc.CreateRefundRequest(ctx, req.PaymentId, req.Reason)
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.CreateRefundRequestResponse{RefundRequestId: refundID, Payment: toPB(p)}, nil
}

func (h *Handler) RefundPayment(ctx context.Context, req *paymentv1.RefundPaymentRequest) (*paymentv1.RefundPaymentResponse, error) {
	p, err := h.uc.Refund(ctx, req.PaymentId, req.Reason)
	if err != nil {
		if err == domain.ErrInvalidTransition {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &paymentv1.RefundPaymentResponse{Payment: toPB(p)}, nil
}
