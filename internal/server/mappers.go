package server

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	codevaldorg "github.com/aosanya/CodeValdOrg"
)

// ToGRPCError is exported for testing. All production code should use toGRPCError.
func ToGRPCError(err error) error { return toGRPCError(err) }

// toGRPCError is the sole translator from domain sentinel errors to gRPC
// status codes. All other layers return raw sentinels.
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	// OAuth 2.0 endpoint errors
	case errors.Is(err, codevaldorg.ErrInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, codevaldorg.ErrInvalidClient):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, codevaldorg.ErrInvalidGrant):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, codevaldorg.ErrUnauthorizedClient):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, codevaldorg.ErrUnsupportedGrantType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, codevaldorg.ErrInvalidScope):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, codevaldorg.ErrAccessDenied):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, codevaldorg.ErrTemporarilyUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, codevaldorg.ErrRateLimitExceeded):
		return status.Error(codes.ResourceExhausted, err.Error())

	// Admin surface — not found
	case errors.Is(err, codevaldorg.ErrOrgNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, codevaldorg.ErrUserNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, codevaldorg.ErrRoleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, codevaldorg.ErrScopeNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, codevaldorg.ErrMembershipNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, codevaldorg.ErrInvitationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, codevaldorg.ErrOAuthClientNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, codevaldorg.ErrEntityNotFound):
		return status.Error(codes.NotFound, err.Error())

	// Admin surface — already exists
	case errors.Is(err, codevaldorg.ErrOrgAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, codevaldorg.ErrUserAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, codevaldorg.ErrRoleAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, codevaldorg.ErrScopeNameCollision):
		return status.Error(codes.AlreadyExists, err.Error())

	// Admin surface — failed preconditions
	case errors.Is(err, codevaldorg.ErrOrgDisabled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, codevaldorg.ErrUserSuspended):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, codevaldorg.ErrRoleBuiltinImmutable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, codevaldorg.ErrInvitationExpired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, codevaldorg.ErrInvitationAlreadyAccepted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, codevaldorg.ErrSuperAdminRequired):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, codevaldorg.ErrImmutableType):
		return status.Error(codes.FailedPrecondition, err.Error())

	// Permission denied
	case errors.Is(err, codevaldorg.ErrScopeReserved):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, codevaldorg.ErrTokenRevoked):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, codevaldorg.ErrTokenExpired):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, codevaldorg.ErrPKCEMismatch):
		return status.Error(codes.PermissionDenied, err.Error())

	// Invalid argument
	case errors.Is(err, codevaldorg.ErrRedirectURIMismatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, codevaldorg.ErrPKCERequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, codevaldorg.ErrPKCEMethodInvalid):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		return status.Error(codes.Internal, "internal error")
	}
}
