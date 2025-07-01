# Event contract definitions for AI enrichment events
# Uses generated protobufs for strong typing

from common.v1 import orchestration_pb2


class EnrichmentEvent:
    """
    Wrapper for incoming enrichment events (from Nexus/Redis Streams).
    """
    def __init__(self, raw_bytes: bytes):
        # Parse as protobuf (OrchestrationEvent or custom event)
        self.proto = orchestration_pb2.OrchestrationEvent()
        self.proto.ParseFromString(raw_bytes)

    @property
    def metadata(self):
        """Return the event's metadata (strongly typed protobuf)."""
        return self.proto.metadata

    @property
    def payload(self):
        """Return the event's payload (strongly typed protobuf, may be model input/output)."""
        return self.proto.payload

    @property
    def model_info(self):
        """Return model info if present (for AI model orchestration)."""
        # Example: orchestration_pb2.OrchestrationEvent may have model fields
        return getattr(self.proto, "model", None)

    @property
    def event_type(self):
        return self.proto.event_type

    @property
    def event_id(self):
        return self.proto.event_id

    def __repr__(self):
        return f"<EnrichmentEvent type={self.event_type} id={self.event_id}>"
