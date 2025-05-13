package server

import (
	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	"google.golang.org/grpc"
)

// RegisterAllServices registers all gRPC services with the server.
func RegisterAllServices(grpcServer *grpc.Server, provider interface {
	User() userpb.UserServiceServer
	Notification() notificationpb.NotificationServiceServer
	Referrals() referralpb.ReferralServiceServer
	Nexus() nexuspb.NexusServiceServer
	Localization() localizationpb.LocalizationServiceServer
	Search() searchpb.SearchServiceServer
	Commerce() commercepb.CommerceServiceServer
},
) {
	userpb.RegisterUserServiceServer(grpcServer, provider.User())
	notificationpb.RegisterNotificationServiceServer(grpcServer, provider.Notification())
	referralpb.RegisterReferralServiceServer(grpcServer, provider.Referrals())
	nexuspb.RegisterNexusServiceServer(grpcServer, provider.Nexus())
	localizationpb.RegisterLocalizationServiceServer(grpcServer, provider.Localization())
	searchpb.RegisterSearchServiceServer(grpcServer, provider.Search())
	commercepb.RegisterCommerceServiceServer(grpcServer, provider.Commerce())
	// TODO: Register MediaService when provider.Media() is implemented
	// TODO: Register ProductService when provider.Product() is implemented
	// TODO: Register TalentService when provider.Talent() is implemented
	// TODO: Register SchedulerService when provider.Scheduler() is implemented
}
