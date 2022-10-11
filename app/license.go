// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-server/v6/app/request"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/product"
	"github.com/mattermost/mattermost-server/v6/store"
)

const (
	LicenseEnv                = "MM_LICENSE"
	LicenseRenewalURL         = "https://customers.mattermost.com/subscribe/renew"
	JWTDefaultTokenExpiration = 7 * 24 * time.Hour // 7 days of expiration
)

var RequestTrialURL = "https://customers.mattermost.com/api/v1/trials"

// ensure the license service wrapper implements `product.LicenseService`
var _ product.LicenseService = (*licenseWrapper)(nil)

// licenseWrapper is an adapter struct that only exposes the
// config related functionality to be passed down to other products.
type licenseWrapper struct {
	srv *Server
}

func (w *licenseWrapper) Name() ServiceKey {
	return LicenseKey
}

func (w *licenseWrapper) GetLicense(c request.CTX) *model.License {
	return w.srv.License(c)
}

func (w *licenseWrapper) RequestTrialLicense(c request.CTX, requesterID string, users int, termsAccepted bool, receiveEmailsAccepted bool) *model.AppError {
	if *w.srv.platform.Config().ExperimentalSettings.RestrictSystemAdmin {
		return model.NewAppError("RequestTrialLicense", "api.restricted_system_admin", nil, "", http.StatusForbidden)
	}

	if !termsAccepted {
		return model.NewAppError("RequestTrialLicense", "api.license.request-trial.bad-request.terms-not-accepted", nil, "", http.StatusBadRequest)
	}

	if users == 0 {
		return model.NewAppError("RequestTrialLicense", "api.license.request-trial.bad-request", nil, "", http.StatusBadRequest)
	}

	requester, err := w.srv.userService.GetUser(requesterID)
	if err != nil {
		var nfErr *store.ErrNotFound
		switch {
		case errors.As(err, &nfErr):
			return model.NewAppError("RequestTrialLicense", MissingAccountError, nil, "", http.StatusNotFound).Wrap(err)
		default:
			return model.NewAppError("RequestTrialLicense", "app.user.get_by_username.app_error", nil, "", http.StatusInternalServerError).Wrap(err)
		}
	}

	trialLicenseRequest := &model.TrialLicenseRequest{
		ServerID:              w.srv.TelemetryId(),
		Name:                  requester.GetDisplayName(model.ShowFullName),
		Email:                 requester.Email,
		SiteName:              *w.srv.platform.Config().TeamSettings.SiteName,
		SiteURL:               *w.srv.platform.Config().ServiceSettings.SiteURL,
		Users:                 users,
		TermsAccepted:         termsAccepted,
		ReceiveEmailsAccepted: receiveEmailsAccepted,
	}

	return w.srv.platform.RequestTrialLicense(c, trialLicenseRequest)
}

// JWTClaims custom JWT claims with the needed information for the
// renewal process
type JWTClaims struct {
	LicenseID   string `json:"license_id"`
	ActiveUsers int64  `json:"active_users"`
	jwt.StandardClaims
}

func (s *Server) License(c request.CTX) *model.License {
	return s.platform.License(c)
}

func (s *Server) LoadLicense(c request.CTX) {
	s.platform.LoadLicense(c)
}

func (s *Server) SaveLicense(c request.CTX, licenseBytes []byte) (*model.License, *model.AppError) {
	return s.platform.SaveLicense(c, licenseBytes)
}

func (s *Server) SetLicense(c request.CTX, license *model.License) bool {
	return s.platform.SetLicense(c, license)
}

func (s *Server) ValidateAndSetLicenseBytes(c request.CTX, b []byte) bool {
	return s.platform.ValidateAndSetLicenseBytes(c, b)
}

func (s *Server) SetClientLicense(m map[string]string) {
	s.platform.SetClientLicense(m)
}

func (s *Server) ClientLicense() map[string]string {
	return s.platform.ClientLicense()
}

func (s *Server) RemoveLicense(c request.CTX) *model.AppError {
	return s.platform.RemoveLicense(c)
}

func (s *Server) AddLicenseListener(listener func(oldLicense, newLicense *model.License)) string {
	return s.platform.AddLicenseListener(listener)
}

func (s *Server) RemoveLicenseListener(id string) {
	s.platform.RemoveLicenseListener(id)
}

func (s *Server) GetSanitizedClientLicense() map[string]string {
	return s.platform.GetSanitizedClientLicense()
}

// GenerateRenewalToken returns a renewal token that expires after duration expiration
func (s *Server) GenerateRenewalToken(c request.CTX, expiration time.Duration) (string, *model.AppError) {
	return s.platform.GenerateRenewalToken(c, expiration)
}

// GenerateLicenseRenewalLink returns a link that points to the CWS where clients can renew license
func (s *Server) GenerateLicenseRenewalLink(c request.CTX) (string, string, *model.AppError) {
	return s.platform.GenerateLicenseRenewalLink(c)
}
