package server

import (
	adminpb "github.com/nmxmxh/master-ovasabi/api/protos/admin/v1"
	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	commercepb "github.com/nmxmxh/master-ovasabi/api/protos/commerce/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	localizationpb "github.com/nmxmxh/master-ovasabi/api/protos/localization/v1"
	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	nexuspb "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	productpb "github.com/nmxmxh/master-ovasabi/api/protos/product/v1"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	schedulerpb "github.com/nmxmxh/master-ovasabi/api/protos/scheduler/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	securitypb "github.com/nmxmxh/master-ovasabi/api/protos/security/v1"
	talentpb "github.com/nmxmxh/master-ovasabi/api/protos/talent/v1"
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
	Media() mediapb.MediaServiceServer
	Product() productpb.ProductServiceServer
	Talent() talentpb.TalentServiceServer
	Scheduler() schedulerpb.SchedulerServiceServer
	Content() contentpb.ContentServiceServer
	Analytics() analyticspb.AnalyticsServiceServer
	ContentModeration() contentmoderationpb.ContentModerationServiceServer
	Messaging() messagingpb.MessagingServiceServer
	Security() securitypb.SecurityServiceServer
	Admin() adminpb.AdminServiceServer
},
) {
	userpb.RegisterUserServiceServer(grpcServer, provider.User())
	notificationpb.RegisterNotificationServiceServer(grpcServer, provider.Notification())
	referralpb.RegisterReferralServiceServer(grpcServer, provider.Referrals())
	nexuspb.RegisterNexusServiceServer(grpcServer, provider.Nexus())
	localizationpb.RegisterLocalizationServiceServer(grpcServer, provider.Localization())
	searchpb.RegisterSearchServiceServer(grpcServer, provider.Search())
	commercepb.RegisterCommerceServiceServer(grpcServer, provider.Commerce())
	mediapb.RegisterMediaServiceServer(grpcServer, provider.Media())
	productpb.RegisterProductServiceServer(grpcServer, provider.Product())
	talentpb.RegisterTalentServiceServer(grpcServer, provider.Talent())
	schedulerpb.RegisterSchedulerServiceServer(grpcServer, provider.Scheduler())
	contentpb.RegisterContentServiceServer(grpcServer, provider.Content())
	analyticspb.RegisterAnalyticsServiceServer(grpcServer, provider.Analytics())
	contentmoderationpb.RegisterContentModerationServiceServer(grpcServer, provider.ContentModeration())
	messagingpb.RegisterMessagingServiceServer(grpcServer, provider.Messaging())
	securitypb.RegisterSecurityServiceServer(grpcServer, provider.Security())
	adminpb.RegisterAdminServiceServer(grpcServer, provider.Admin())
}
