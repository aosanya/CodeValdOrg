package server_test

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	codevaldorg "github.com/aosanya/CodeValdOrg"
	"github.com/aosanya/CodeValdOrg/internal/server"
)

func TestToGRPCError_NilReturnsNil(t *testing.T) {
	if err := server.ToGRPCError(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func mapperCase(t *testing.T, input error, wantCode codes.Code) {
	t.Helper()
	err := server.ToGRPCError(input)
	if err == nil {
		t.Fatalf("expected error for input %v", input)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status, got %v", err)
	}
	if st.Code() != wantCode {
		t.Errorf("input %v: got code %v, want %v", input, st.Code(), wantCode)
	}
}

func TestToGRPCError_Mappings(t *testing.T) {
	cases := []struct {
		err      error
		wantCode codes.Code
	}{
		{codevaldorg.ErrInvalidRequest, codes.InvalidArgument},
		{codevaldorg.ErrInvalidClient, codes.Unauthenticated},
		{codevaldorg.ErrInvalidGrant, codes.PermissionDenied},
		{codevaldorg.ErrUnauthorizedClient, codes.PermissionDenied},
		{codevaldorg.ErrUnsupportedGrantType, codes.InvalidArgument},
		{codevaldorg.ErrInvalidScope, codes.InvalidArgument},
		{codevaldorg.ErrAccessDenied, codes.PermissionDenied},
		{codevaldorg.ErrTemporarilyUnavailable, codes.Unavailable},
		{codevaldorg.ErrRateLimitExceeded, codes.ResourceExhausted},

		{codevaldorg.ErrOrgNotFound, codes.NotFound},
		{codevaldorg.ErrUserNotFound, codes.NotFound},
		{codevaldorg.ErrRoleNotFound, codes.NotFound},
		{codevaldorg.ErrScopeNotFound, codes.NotFound},
		{codevaldorg.ErrMembershipNotFound, codes.NotFound},
		{codevaldorg.ErrInvitationNotFound, codes.NotFound},
		{codevaldorg.ErrOAuthClientNotFound, codes.NotFound},

		{codevaldorg.ErrOrgAlreadyExists, codes.AlreadyExists},
		{codevaldorg.ErrUserAlreadyExists, codes.AlreadyExists},
		{codevaldorg.ErrRoleAlreadyExists, codes.AlreadyExists},
		{codevaldorg.ErrScopeNameCollision, codes.AlreadyExists},

		{codevaldorg.ErrOrgDisabled, codes.FailedPrecondition},
		{codevaldorg.ErrUserSuspended, codes.FailedPrecondition},
		{codevaldorg.ErrRoleBuiltinImmutable, codes.FailedPrecondition},
		{codevaldorg.ErrInvitationExpired, codes.FailedPrecondition},
		{codevaldorg.ErrInvitationAlreadyAccepted, codes.FailedPrecondition},
		{codevaldorg.ErrSuperAdminRequired, codes.FailedPrecondition},
		{codevaldorg.ErrImmutableType, codes.FailedPrecondition},

		{codevaldorg.ErrScopeReserved, codes.PermissionDenied},
		{codevaldorg.ErrTokenRevoked, codes.PermissionDenied},
		{codevaldorg.ErrTokenExpired, codes.PermissionDenied},
		{codevaldorg.ErrPKCEMismatch, codes.PermissionDenied},

		{codevaldorg.ErrRedirectURIMismatch, codes.InvalidArgument},
		{codevaldorg.ErrPKCERequired, codes.InvalidArgument},
		{codevaldorg.ErrPKCEMethodInvalid, codes.InvalidArgument},
	}

	for _, c := range cases {
		t.Run(c.err.Error(), func(t *testing.T) {
			mapperCase(t, c.err, c.wantCode)
		})
	}
}

func TestToGRPCError_UnknownReturnsInternal(t *testing.T) {
	import_error := codevaldorg.ErrOrgNotFound // just pick any sentinel to confirm fallthrough doesn't happen
	_ = import_error
	// Use an arbitrary error not in the catalog
	unknown := &unexportedErr{"something went wrong"}
	mapperCase(t, unknown, codes.Internal)
}

type unexportedErr struct{ msg string }

func (e *unexportedErr) Error() string { return e.msg }
