package server

import (
	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v0"
	authpb "github.com/nmxmxh/master-ovasabi/api/protos/auth/v0"
	babelpb "github.com/nmxmxh/master-ovasabi/api/protos/babel/v0"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	financepb "github.com/nmxmxh/master-ovasabi/api/protos/finance/v0"
	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v0"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v0"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v0"
	"github.com/nmxmxh/master-ovasabi/internal/service"
	"google.golang.org/grpc"
)

func RegisterAllServices(grpcServer *grpc.Server, provider *service.Provider) {
	authpb.RegisterAuthServiceServer(grpcServer, provider.Auth())
	userpb.RegisterUserServiceServer(grpcServer, provider.User())
	notificationpb.RegisterNotificationServiceServer(grpcServer, provider.Notification())
	broadcastpb.RegisterBroadcastServiceServer(grpcServer, provider.Broadcast())
	i18npb.RegisterI18NServiceServer(grpcServer, provider.I18n())
	quotespb.RegisterQuotesServiceServer(grpcServer, provider.Quotes())
	referralpb.RegisterReferralServiceServer(grpcServer, provider.Referrals())
	assetpb.RegisterAssetServiceServer(grpcServer, provider.Asset())
	financepb.RegisterFinanceServiceServer(grpcServer, provider.Finance())
	nexuspb.RegisterNexusServiceServer(grpcServer, provider.Nexus())
	babelpb.RegisterBabelServiceServer(grpcServer, provider.Babel())
}
