package vespiary

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vx-labs/vespiary/vespiary/state"
	"github.com/vx-labs/wasp/v4/wasp/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultMountPoint = "_default"

func fingerprintBytes(buf []byte) string {
	sum := sha256.Sum256(buf)
	return fmt.Sprintf("%x", sum)
}

func fingerprintString(buf string) string {
	return fingerprintBytes([]byte(buf))
}

type WaspAuthenticationServer struct {
	fsm            FSM
	state          state.Store
	trustedAdminCA x509.VerifyOptions
}

func NewWaspAuthenticationServer(fsm FSM, state state.Store, trustedAdminCAPool *x509.CertPool) *WaspAuthenticationServer {
	return &WaspAuthenticationServer{
		fsm:   fsm,
		state: state,
		trustedAdminCA: x509.VerifyOptions{
			Roots: trustedAdminCAPool,
		},
	}
}

func (s *WaspAuthenticationServer) Serve(grpcServer *grpc.Server) {
	auth.RegisterAuthenticationServer(grpcServer, s)
}

func (s *WaspAuthenticationServer) AuthenticateMQTTClient(ctx context.Context, input *auth.WaspAuthenticationRequest) (*auth.WaspAuthenticationResponse, error) {
	if len(input.Transport.X509CertificateChain) > 0 {
		cert, err := x509.ParseCertificate(input.Transport.X509CertificateChain[0])
		if err == nil {
			_, err := cert.Verify(s.trustedAdminCA)
			if err == nil {
				return &auth.WaspAuthenticationResponse{
					ID:         fmt.Sprintf("_admin/%s", uuid.New().String()),
					MountPoint: "_root",
				}, nil
			}
		}
	}
	tokens := strings.SplitN(string(input.MQTT.Username), "/", 3)
	if len(tokens) == 3 {
		account, err := s.state.Accounts().ByName(string(tokens[0]))
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid username or password")
		}
		application, err := s.state.Applications().ByNameAndAccountID(tokens[1], account.ID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid username or password")
		}
		profile, err := s.state.ApplicationProfiles().ByNameAndApplicationID(tokens[2], application.ID)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid username or password")
		}
		candidatePassword := sha256.Sum256(append(input.MQTT.Password, profile.PasswordSalt...))

		if bytes.Equal(candidatePassword[:], profile.PasswordFingerprint) {
			return &auth.WaspAuthenticationResponse{
				ID:         fmt.Sprintf("%s/%s", profile.ID, uuid.New().String()),
				MountPoint: fmt.Sprintf("_root/%s/%s", account.ID, application.ID),
			}, nil
		}
		return nil, status.Error(codes.InvalidArgument, "invalid username or password")
	}
	account, err := s.state.Accounts().ByDeviceUsername(string(input.MQTT.Username))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid username or password")
	}
	device, err := s.state.DeviceByName(account.ID, string(input.MQTT.ClientID))
	if err != nil {
		s.fsm.CreateDevice(ctx, account.ID, string(input.MQTT.ClientID), fingerprintBytes(input.MQTT.Password), false)
		return nil, status.Error(codes.InvalidArgument, "invalid username or password")
	}
	if device.Active && device.Password == fingerprintBytes(input.MQTT.Password) {
		return &auth.WaspAuthenticationResponse{
			ID:         device.ID,
			MountPoint: device.Owner,
		}, nil
	}
	return nil, status.Error(codes.InvalidArgument, "invalid username or password")
}
