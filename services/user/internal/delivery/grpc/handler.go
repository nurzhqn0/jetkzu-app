package grpc

import (
	"context"

	userv1 "github.com/jetkzu/jetkzu/gen/go/user/v1"
	"github.com/jetkzu/jetkzu/services/user/internal/domain"
	"github.com/jetkzu/jetkzu/services/user/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	userv1.UnimplementedUserServiceServer
	uc *usecase.UserUseCase
}

func NewHandler(uc *usecase.UserUseCase) *Handler { return &Handler{uc: uc} }

func toPB(u *domain.User) *userv1.User {
	return &userv1.User{
		Id:            u.ID,
		Email:         u.Email,
		FullName:      u.FullName,
		Phone:         u.Phone,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		CreatedAt:     timestamppb.New(u.CreatedAt),
		UpdatedAt:     timestamppb.New(u.UpdatedAt),
	}
}

func (h *Handler) RegisterUser(ctx context.Context, req *userv1.RegisterUserRequest) (*userv1.RegisterUserResponse, error) {
	res, err := h.uc.Register(ctx, usecase.RegisterInput{
		Email: req.Email, Password: req.Password, FullName: req.FullName, Phone: req.Phone, Role: req.Role,
	})
	if err != nil {
		if err == domain.ErrEmailAlreadyTaken {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &userv1.RegisterUserResponse{User: toPB(res.User), VerificationToken: res.VerificationToken}, nil
}

func (h *Handler) LoginUser(ctx context.Context, req *userv1.LoginUserRequest) (*userv1.LoginUserResponse, error) {
	res, err := h.uc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if err == domain.ErrInvalidCredential {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.LoginUserResponse{
		AccessToken: res.Token,
		ExpiresAt:   timestamppb.New(res.ExpiresAt),
		User:        toPB(res.User),
	}, nil
}

func (h *Handler) LogoutUser(ctx context.Context, req *userv1.LogoutUserRequest) (*userv1.LogoutUserResponse, error) {
	ok, err := h.uc.Logout(ctx, req.UserId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.LogoutUserResponse{Ok: ok}, nil
}

func (h *Handler) ValidateSession(ctx context.Context, req *userv1.ValidateSessionRequest) (*userv1.ValidateSessionResponse, error) {
	u, ok, err := h.uc.ValidateSession(ctx, req.AccessToken)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return &userv1.ValidateSessionResponse{Valid: false}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !ok {
		return &userv1.ValidateSessionResponse{Valid: false}, nil
	}
	return &userv1.ValidateSessionResponse{Valid: true, User: toPB(u)}, nil
}

func (h *Handler) GetUserProfile(ctx context.Context, req *userv1.GetUserProfileRequest) (*userv1.GetUserProfileResponse, error) {
	u, err := h.uc.GetProfile(ctx, req.UserId)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.GetUserProfileResponse{User: toPB(u)}, nil
}

func (h *Handler) GetUserByEmail(ctx context.Context, req *userv1.GetUserByEmailRequest) (*userv1.GetUserByEmailResponse, error) {
	u, err := h.uc.GetByEmail(ctx, req.Email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &userv1.GetUserByEmailResponse{User: toPB(u)}, nil
}

func (h *Handler) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	users, err := h.uc.List(ctx, req.Role, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	out := make([]*userv1.User, 0, len(users))
	for _, u := range users {
		out = append(out, toPB(u))
	}
	return &userv1.ListUsersResponse{Users: out}, nil
}

func (h *Handler) UpdateUserProfile(ctx context.Context, req *userv1.UpdateUserProfileRequest) (*userv1.UpdateUserProfileResponse, error) {
	u, err := h.uc.UpdateProfile(ctx, usecase.UpdateInput{
		UserID: req.UserId, FullName: req.FullName, Phone: req.Phone,
	})
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.UpdateUserProfileResponse{User: toPB(u)}, nil
}

func (h *Handler) ChangePassword(ctx context.Context, req *userv1.ChangePasswordRequest) (*userv1.ChangePasswordResponse, error) {
	ok, err := h.uc.ChangePassword(ctx, req.UserId, req.OldPassword, req.NewPassword)
	if err != nil {
		if err == domain.ErrInvalidCredential {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &userv1.ChangePasswordResponse{Changed: ok}, nil
}

func (h *Handler) ResetPassword(ctx context.Context, req *userv1.ResetPasswordRequest) (*userv1.ResetPasswordResponse, error) {
	ok, err := h.uc.ResetPassword(ctx, req.Email, req.NewPassword)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &userv1.ResetPasswordResponse{Reset_: ok}, nil
}

func (h *Handler) VerifyUserEmail(ctx context.Context, req *userv1.VerifyUserEmailRequest) (*userv1.VerifyUserEmailResponse, error) {
	ok, err := h.uc.VerifyEmail(ctx, req.UserId, req.Token)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.VerifyUserEmailResponse{Verified: ok}, nil
}

func (h *Handler) ResendVerification(ctx context.Context, req *userv1.ResendVerificationRequest) (*userv1.ResendVerificationResponse, error) {
	token, err := h.uc.ResendVerification(ctx, req.UserId)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.ResendVerificationResponse{VerificationToken: token}, nil
}

func (h *Handler) DeactivateUser(ctx context.Context, req *userv1.DeactivateUserRequest) (*userv1.DeactivateUserResponse, error) {
	u, err := h.uc.Deactivate(ctx, req.UserId)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &userv1.DeactivateUserResponse{User: toPB(u)}, nil
}

func (h *Handler) UpdateUserRole(ctx context.Context, req *userv1.UpdateUserRoleRequest) (*userv1.UpdateUserRoleResponse, error) {
	u, err := h.uc.UpdateRole(ctx, req.UserId, req.Role)
	if err != nil {
		if err == domain.ErrUserNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &userv1.UpdateUserRoleResponse{User: toPB(u)}, nil
}
