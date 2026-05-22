package grpc

import (
	"context"

	notifv1 "github.com/jetkzu/jetkzu/gen/go/notification/v1"
	"github.com/jetkzu/jetkzu/services/notification/internal/domain"
	"github.com/jetkzu/jetkzu/services/notification/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	notifv1.UnimplementedNotificationServiceServer
	uc *usecase.UseCase
}

func NewHandler(uc *usecase.UseCase) *Handler { return &Handler{uc: uc} }

func toPB(n *domain.Notification) *notifv1.Notification {
	if n == nil {
		return nil
	}
	return &notifv1.Notification{
		Id: n.ID, UserId: n.UserID, Channel: n.Channel,
		Subject: n.Subject, Body: n.Body, Status: n.Status,
		CreatedAt: timestamppb.New(n.CreatedAt), SentAt: timestamppb.New(n.SentAt),
	}
}

func (h *Handler) SendEmailNotification(ctx context.Context, req *notifv1.SendEmailNotificationRequest) (*notifv1.SendEmailNotificationResponse, error) {
	n, err := h.uc.SendEmail(ctx, req.UserId, req.To, req.Subject, req.Body)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &notifv1.SendEmailNotificationResponse{Notification: toPB(n)}, nil
}

func (h *Handler) SendRideReceipt(ctx context.Context, req *notifv1.SendRideReceiptRequest) (*notifv1.SendRideReceiptResponse, error) {
	n, err := h.uc.SendRideReceipt(ctx, req.UserId, req.To, req.RideId, req.PaymentId, req.Amount)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &notifv1.SendRideReceiptResponse{Notification: toPB(n)}, nil
}

func (h *Handler) GetNotificationHistory(ctx context.Context, req *notifv1.GetNotificationHistoryRequest) (*notifv1.GetNotificationHistoryResponse, error) {
	list, err := h.uc.History(ctx, req.UserId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*notifv1.Notification, 0, len(list))
	for _, n := range list {
		out = append(out, toPB(n))
	}
	return &notifv1.GetNotificationHistoryResponse{Notifications: out}, nil
}

func (h *Handler) ListUnreadNotifications(ctx context.Context, req *notifv1.ListUnreadNotificationsRequest) (*notifv1.ListUnreadNotificationsResponse, error) {
	list, err := h.uc.Unread(ctx, req.UserId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	out := make([]*notifv1.Notification, 0, len(list))
	for _, n := range list {
		out = append(out, toPB(n))
	}
	return &notifv1.ListUnreadNotificationsResponse{Notifications: out}, nil
}

func (h *Handler) CountUnreadNotifications(ctx context.Context, req *notifv1.CountUnreadNotificationsRequest) (*notifv1.CountUnreadNotificationsResponse, error) {
	count, err := h.uc.CountUnread(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &notifv1.CountUnreadNotificationsResponse{Count: int32(count)}, nil
}

func (h *Handler) MarkNotificationAsRead(ctx context.Context, req *notifv1.MarkNotificationAsReadRequest) (*notifv1.MarkNotificationAsReadResponse, error) {
	n, err := h.uc.MarkRead(ctx, req.NotificationId)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &notifv1.MarkNotificationAsReadResponse{Notification: toPB(n)}, nil
}

func (h *Handler) MarkAllNotificationsAsRead(ctx context.Context, req *notifv1.MarkAllNotificationsAsReadRequest) (*notifv1.MarkAllNotificationsAsReadResponse, error) {
	updated, err := h.uc.MarkAllRead(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &notifv1.MarkAllNotificationsAsReadResponse{Updated: int32(updated)}, nil
}

func (h *Handler) ResendNotification(ctx context.Context, req *notifv1.ResendNotificationRequest) (*notifv1.ResendNotificationResponse, error) {
	n, err := h.uc.Resend(ctx, req.NotificationId)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &notifv1.ResendNotificationResponse{Notification: toPB(n)}, nil
}

func (h *Handler) DeleteNotification(ctx context.Context, req *notifv1.DeleteNotificationRequest) (*notifv1.DeleteNotificationResponse, error) {
	ok, err := h.uc.Delete(ctx, req.NotificationId)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &notifv1.DeleteNotificationResponse{Deleted: ok}, nil
}
