package bridge

import (
	"context"
	"fmt"

	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
)

type Router struct {
	routingRules []RoutingRule
	log          *zap.Logger
}

type RoutingRule struct {
	Match    MetadataMatcher `json:"match"`
	Protocol string          `json:"protocol"`
	Endpoint string          `json:"endpoint"`
	Priority int             `json:"priority"`
}

type MetadataMatcher map[string]string

func NewRouter(rules []RoutingRule, log *zap.Logger) *Router {
	return &Router{
		routingRules: rules,
		log:          log,
	}
}

// Route routes a message to the correct adapter based on metadata and routing rules.
func (r *Router) Route(ctx context.Context, msg *Message) error {
	log := r.log
	metaMap := make(map[string]interface{}, len(msg.Metadata))
	for k, v := range msg.Metadata {
		metaMap[k] = v
	}
	metaProto := metadata.MapToProto(metaMap)
	if !AuthorizeTransport(ctx, msg.Destination, metaProto) {
		LogTransportEvent(log, "unauthorized", msg)
		return fmt.Errorf("unauthorized")
	}
	LogTransportEvent(log, "route_attempt", msg)

	for _, rule := range r.routingRules {
		if rule.Matches(msg.Metadata) {
			if adapter, ok := GetAdapter(rule.Protocol); ok {
				LogTransportEvent(log, "route_success", msg)
				return adapter.Send(ctx, msg)
			}
		}
	}
	LogTransportEvent(log, "route_no_adapter", msg)
	return fmt.Errorf("no adapter found for message")
}

// RouteAsync routes a message asynchronously and returns a channel for the error result.
func (r *Router) RouteAsync(ctx context.Context, msg *Message) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)
		errCh <- r.Route(ctx, msg)
	}()
	return errCh
}

// Matches checks if the rule's matcher matches the given metadata.
func (r RoutingRule) Matches(meta map[string]string) bool {
	return r.Match.Matches(meta)
}

// Matches checks if the metadata matches the rule's matcher.
func (m MetadataMatcher) Matches(meta map[string]string) bool {
	for k, v := range m {
		if meta[k] != v {
			return false
		}
	}
	return true
}
