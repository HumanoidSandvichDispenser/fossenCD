package service

import (
	"context"
	"net/mail"
	"time"

	"github.com/humanoidsandvichdispenser/fossencd/backend/internal/store"
)

// EmailService manages an account's email set.
type EmailService struct {
	db    *store.UserStore
	users *AuthService

	// requireConfirmation withholds VerifiedAt on added addresses until a
	// confirmation flow resolves it. Off by default: without a mailer, adding
	// an address verifies it immediately, like gitea without SMTP.
	requireConfirmation bool
}

// ListEmails returns a user's email set.
func (s *EmailService) ListEmails(ctx context.Context, userID uint) ([]store.UserEmail, error) {
	return s.db.ListEmails(ctx, userID)
}

// AddEmail adds an address to a user's email set. Returns ErrBadRequest for a
// malformed address, or ErrConflict when it is already registered to any
// account.
func (s *EmailService) AddEmail(ctx context.Context, userID uint, email string) error {
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return ErrBadRequest
	}

	var verifiedAt *time.Time
	if !s.requireConfirmation {
		now := time.Now()
		verifiedAt = &now
	}
	return mapErr(s.db.AddEmail(ctx, userID, email, verifiedAt))
}

// RemoveEmail drops an address from a user's email set. The registration
// (primary) address cannot be removed.
func (s *EmailService) RemoveEmail(ctx context.Context, userID uint, email string) error {
	u, err := s.users.User(ctx, userID)
	if err != nil {
		return err
	}
	if u.Email == email {
		return ErrBadRequest
	}
	return mapErr(s.db.RemoveEmail(ctx, userID, email))
}

// ClaimVerification is the outcome of checking a claimed email against an
// endpoint's owning accounts.
type ClaimVerification struct {
	Verified bool

	// Username of the account the claim verified against, empty otherwise.
	Username string
}

// VerifyClaim checks a peer's claimed email: verified iff some account that
// registered endpointID has email in its verified set. The claim never
// authorizes anything; this only decides the badge.
func (s *EmailService) VerifyClaim(ctx context.Context, endpointID, email string) (ClaimVerification, error) {
	if email == "" {
		return ClaimVerification{}, nil
	}

	u, err := s.db.VerifiedOwnerOfClaim(ctx, endpointID, email)
	if err == store.ErrNotFound {
		return ClaimVerification{}, nil
	}
	if err != nil {
		return ClaimVerification{}, err
	}
	return ClaimVerification{Verified: true, Username: u.Username}, nil
}
