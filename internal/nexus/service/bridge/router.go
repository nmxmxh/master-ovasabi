package bridge

import (
	"context"
	"fmt"
)

type Router struct {
	routingRules []RoutingRule
}

type RoutingRule struct {
	Match    MetadataMatcher `json:"match"`
	Protocol string          `json:"protocol"`
	Endpoint string          `json:"endpoint"`
	Priority int             `json:"priority"`
}

type MetadataMatcher map[string]string

// Route routes a message to the correct adapter based on metadata and routing rules.
func (r *Router) Route(ctx context.Context, msg *Message) error {
	// Enforce RBAC and log audit
	if !AuthorizeTransport("send", msg.Destination, msg.Metadata) {
		LogTransportEvent("unauthorized", msg)
		return fmt.Errorf("unauthorized")
	}
	LogTransportEvent("route_attempt", msg)

	for _, rule := range r.routingRules {
		if rule.Matches(msg.Metadata) {
			if adapter, ok := GetAdapter(rule.Protocol); ok {
				LogTransportEvent("route_success", msg)
				return adapter.Send(ctx, msg)
			}
		}
	}
	LogTransportEvent("route_no_adapter", msg)
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
func (r RoutingRule) Matches(metadata map[string]string) bool {
	return r.Match.Matches(metadata)
}

// Matches checks if the metadata matches the rule's matcher.
func (m MetadataMatcher) Matches(metadata map[string]string) bool {
	for k, v := range m {
		if metadata[k] != v {
			return false
		}
	}
	return true
}
