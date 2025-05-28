package messaging

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrMessageNotFound      = errors.New("message not found")
	ErrThreadNotFound       = errors.New("thread not found")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrChatGroupNotFound    = errors.New("chat group not found")
)

var logInstance *zap.Logger

// Message represents a message in the messaging system.
type Message struct {
	ID             string             `db:"id"`
	MasterID       int64              `db:"master_id"`
	MasterUUID     string             `db:"master_uuid"`
	ThreadID       string             `db:"thread_id"`
	ConversationID string             `db:"conversation_id"`
	ChatGroupID    string             `db:"chat_group_id"`
	SenderID       string             `db:"sender_id"`
	RecipientIDs   []string           `db:"recipient_ids"`
	Content        string             `db:"content"`
	Type           string             `db:"type"`
	Attachments    []byte             `db:"attachments"`
	Reactions      []byte             `db:"reactions"`
	Status         string             `db:"status"`
	Edited         bool               `db:"edited"`
	Deleted        bool               `db:"deleted"`
	Metadata       *commonpb.Metadata `db:"metadata"`
	CreatedAt      time.Time          `db:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"`
	CampaignID     int64              `db:"campaign_id"`
}

// Thread represents a messaging thread.
type Thread struct {
	ID             string             `db:"id"`
	MasterID       int64              `db:"master_id"`
	MasterUUID     string             `db:"master_uuid"`
	ParticipantIDs []string           `db:"participant_ids"`
	Subject        string             `db:"subject"`
	MessageIDs     []string           `db:"message_ids"`
	Metadata       *commonpb.Metadata `db:"metadata"`
	CreatedAt      time.Time          `db:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"`
	CampaignID     int64              `db:"campaign_id"`
}

// Conversation represents a messaging conversation.
type Conversation struct {
	ID             string             `db:"id"`
	MasterID       int64              `db:"master_id"`
	MasterUUID     string             `db:"master_uuid"`
	ParticipantIDs []string           `db:"participant_ids"`
	ChatGroupID    string             `db:"chat_group_id"`
	ThreadIDs      []string           `db:"thread_ids"`
	Metadata       *commonpb.Metadata `db:"metadata"`
	CreatedAt      time.Time          `db:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"`
	CampaignID     int64              `db:"campaign_id"`
}

// ChatGroup represents a group chat.
type ChatGroup struct {
	ID          string             `db:"id"`
	MasterID    int64              `db:"master_id"`
	MasterUUID  string             `db:"master_uuid"`
	Name        string             `db:"name"`
	Description string             `db:"description"`
	MemberIDs   []string           `db:"member_ids"`
	Roles       map[string]string  `db:"roles"`
	Metadata    *commonpb.Metadata `db:"metadata"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
	CampaignID  int64              `db:"campaign_id"`
}

// MessageEvent represents an event for analytics/audit.
type MessageEvent struct {
	ID         string    `db:"id"`
	MasterID   int64     `db:"master_id"`
	MasterUUID string    `db:"master_uuid"`
	MessageID  string    `db:"message_id"`
	UserID     string    `db:"user_id"`
	EventType  string    `db:"event_type"`
	Payload    []byte    `db:"payload"`
	CreatedAt  time.Time `db:"created_at"`
}

// MessagingRepository handles operations on the messaging tables.
type Repository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

func NewRepository(db *sql.DB, masterRepo repository.MasterRepository) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db),
		masterRepo:     masterRepo,
	}
}

// safeServiceSpecific returns meta.ServiceSpecific if non-nil, else an empty structpb.Struct.
func safeServiceSpecific(meta *commonpb.Metadata) *structpb.Struct {
	if meta != nil && meta.ServiceSpecific != nil {
		return meta.ServiceSpecific
	}
	return &structpb.Struct{Fields: map[string]*structpb.Value{}}
}

// CreateMessage inserts a new message record.
func (r *Repository) CreateMessage(ctx context.Context, msg *Message) (*Message, error) {
	masterName := r.GenerateMasterName(repository.EntityType("message"), msg.Content, msg.Type, fmt.Sprintf("sender-%s", msg.SenderID))
	masterID, err := r.masterRepo.Create(ctx, repository.EntityType("message"), masterName)
	if err != nil {
		return nil, err
	}
	msg.MasterID = masterID
	var metadataJSON []byte
	if msg.Metadata != nil {
		metadataJSON, err = protojson.Marshal(msg.Metadata)
		if err != nil {
			return nil, err
		}
	}
	query := `INSERT INTO service_messaging_main (
		master_id, thread_id, conversation_id, chat_group_id, sender_id, recipient_ids, content, type, attachments, reactions, status, edited, deleted, metadata, campaign_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW()
	) RETURNING id, created_at, updated_at`
	err = r.GetDB().QueryRowContext(ctx, query,
		msg.MasterID, msg.ThreadID, msg.ConversationID, msg.ChatGroupID, msg.SenderID, pq.Array(msg.RecipientIDs), msg.Content, msg.Type, msg.Attachments, msg.Reactions, msg.Status, msg.Edited, msg.Deleted, metadataJSON, msg.CampaignID,
	).Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)
	if err != nil {
		if err := r.masterRepo.Delete(ctx, msg.MasterID); err != nil {
			if logInstance != nil {
				logInstance.Error("failed to delete master record", zap.Error(err))
			}
		}
		return nil, err
	}
	return msg, nil
}

// GetMessage retrieves a message by ID.
func (r *Repository) GetMessage(ctx context.Context, id string) (*Message, error) {
	msg := &Message{}
	var metadataStr string
	query := `SELECT id, master_id, thread_id, conversation_id, chat_group_id, sender_id, recipient_ids, content, type, attachments, reactions, status, edited, deleted, metadata, campaign_id, created_at, updated_at FROM service__main WHERE id = $1`
	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(
		&msg.ID, &msg.MasterID, &msg.ThreadID, &msg.ConversationID, &msg.ChatGroupID, &msg.SenderID, pq.Array(&msg.RecipientIDs), &msg.Content, &msg.Type, &msg.Attachments, &msg.Reactions, &msg.Status, &msg.Edited, &msg.Deleted, &metadataStr, &msg.CampaignID, &msg.CreatedAt, &msg.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	if metadataStr != "" {
		msg.Metadata = &commonpb.Metadata{}
		err := protojson.Unmarshal([]byte(metadataStr), msg.Metadata)
		if err != nil {
			logInstance.Warn("failed to unmarshal message metadata", zap.Error(err))
		}
	} else {
		msg.Metadata = &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": "metadata unavailable", "message_id": msg.ID}, safeServiceSpecific(msg.Metadata)),
			Tags:            []string{},
			Features:        []string{},
		}
	}
	return msg, nil
}

// UpdateMessage updates a message record.
func (r *Repository) UpdateMessage(ctx context.Context, msg *Message) error {
	var metadataJSON []byte
	var err error
	if msg.Metadata != nil {
		metadataJSON, err = protojson.Marshal(msg.Metadata)
		if err != nil {
			return err
		}
	}
	query := `UPDATE service_messaging_main SET thread_id=$1, conversation_id=$2, chat_group_id=$3, sender_id=$4, recipient_ids=$5, content=$6, type=$7, attachments=$8, reactions=$9, status=$10, edited=$11, deleted=$12, metadata=$13, updated_at=NOW() WHERE id=$14`
	result, err := r.GetDB().ExecContext(ctx, query,
		msg.ThreadID, msg.ConversationID, msg.ChatGroupID, msg.SenderID, pq.Array(msg.RecipientIDs), msg.Content, msg.Type, msg.Attachments, msg.Reactions, msg.Status, msg.Edited, msg.Deleted, metadataJSON, msg.ID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMessageNotFound
	}
	return nil
}

// DeleteMessage removes a message and its master record.
func (r *Repository) DeleteMessage(ctx context.Context, id string) error {
	msg, err := r.GetMessage(ctx, id)
	if err != nil {
		return err
	}
	if err := r.masterRepo.Delete(ctx, msg.MasterID); err != nil {
		if logInstance != nil {
			logInstance.Error("failed to delete master record", zap.Error(err))
		}
	}
	return nil
}

// ListMessages retrieves a paginated list of messages for a thread or conversation.
func (r *Repository) ListMessages(ctx context.Context, threadID, conversationID string, limit, offset int) ([]*Message, error) {
	query := `SELECT id, master_id, thread_id, conversation_id, chat_group_id, sender_id, recipient_ids, content, type, attachments, reactions, status, edited, deleted, metadata, campaign_id, created_at, updated_at FROM service_messaging_main WHERE (thread_id = $1 OR conversation_id = $2) ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.GetDB().QueryContext(ctx, query, threadID, conversationID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var metadataStr string
		err := rows.Scan(&msg.ID, &msg.MasterID, &msg.ThreadID, &msg.ConversationID, &msg.ChatGroupID, &msg.SenderID, pq.Array(&msg.RecipientIDs), &msg.Content, &msg.Type, &msg.Attachments, &msg.Reactions, &msg.Status, &msg.Edited, &msg.Deleted, &metadataStr, &msg.CampaignID, &msg.CreatedAt, &msg.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if metadataStr != "" {
			msg.Metadata = &commonpb.Metadata{}
			err := protojson.Unmarshal([]byte(metadataStr), msg.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal message metadata", zap.Error(err))
				return nil, err
			}
		} else {
			msg.Metadata = &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": "metadata unavailable", "message_id": msg.ID}, safeServiceSpecific(msg.Metadata)),
				Tags:            []string{},
				Features:        []string{},
			}
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// ListThreads retrieves a paginated list of threads.
func (r *Repository) ListThreads(ctx context.Context, limit, offset int) ([]*Thread, error) {
	query := `SELECT id, master_id, subject, participant_ids, message_ids, metadata, campaign_id, created_at, updated_at FROM service_messaging_thread ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.GetDB().QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var threads []*Thread
	for rows.Next() {
		thread := &Thread{}
		var metadataStr string
		err := rows.Scan(&thread.ID, &thread.MasterID, &thread.Subject, pq.Array(&thread.ParticipantIDs), pq.Array(&thread.MessageIDs), &metadataStr, &thread.CampaignID, &thread.CreatedAt, &thread.UpdatedAt)
		if err != nil {
			return nil, err
		}
		thread.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), thread.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal thread metadata", zap.Error(err))
				return nil, err
			}
		}
		threads = append(threads, thread)
	}
	return threads, rows.Err()
}

// ListConversations retrieves a paginated list of conversations.
func (r *Repository) ListConversations(ctx context.Context, limit, offset int) ([]*Conversation, error) {
	query := `SELECT id, master_id, participant_ids, chat_group_id, thread_ids, metadata, campaign_id, created_at, updated_at FROM service_messaging_conversation ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.GetDB().QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conversations []*Conversation
	for rows.Next() {
		conv := &Conversation{}
		var metadataStr string
		err := rows.Scan(&conv.ID, &conv.MasterID, pq.Array(&conv.ParticipantIDs), &conv.ChatGroupID, pq.Array(&conv.ThreadIDs), &metadataStr, &conv.CampaignID, &conv.CreatedAt, &conv.UpdatedAt)
		if err != nil {
			return nil, err
		}
		conv.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), conv.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal conversation metadata", zap.Error(err))
				return nil, err
			}
		}
		conversations = append(conversations, conv)
	}
	return conversations, rows.Err()
}

// ListChatGroups retrieves a paginated list of chat groups.
func (r *Repository) ListChatGroups(ctx context.Context, limit, offset int) ([]*ChatGroup, error) {
	query := `SELECT id, master_id, name, description, member_ids, roles, metadata, campaign_id, created_at, updated_at FROM service_messaging_chat_group ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.GetDB().QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []*ChatGroup
	for rows.Next() {
		group := &ChatGroup{}
		var metadataStr, rolesStr string
		err := rows.Scan(&group.ID, &group.MasterID, &group.Name, &group.Description, pq.Array(&group.MemberIDs), &rolesStr, &metadataStr, &group.CampaignID, &group.CreatedAt, &group.UpdatedAt)
		if err != nil {
			return nil, err
		}
		group.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), group.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal chat group metadata", zap.Error(err))
				return nil, err
			}
		}
		group.Roles = map[string]string{}
		if rolesStr != "" {
			if err := json.Unmarshal([]byte(rolesStr), &group.Roles); err != nil {
				if logInstance != nil {
					logInstance.Warn("failed to unmarshal roles", zap.Error(err))
				}
			}
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// ListMessageEvents retrieves a paginated list of message events for analytics/audit.
func (r *Repository) ListMessageEvents(ctx context.Context, messageID string, limit, offset int) ([]*MessageEvent, error) {
	query := `SELECT id, master_id, message_id, user_id, event_type, payload, created_at FROM service_messaging_event WHERE message_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.GetDB().QueryContext(ctx, query, messageID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*MessageEvent
	for rows.Next() {
		event := &MessageEvent{}
		err := rows.Scan(&event.ID, &event.MasterID, &event.MessageID, &event.UserID, &event.EventType, &event.Payload, &event.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// ListMessageEventsByUser retrieves a paginated list of message events for a user.
func (r *Repository) ListMessageEventsByUser(ctx context.Context, userID string, limit, offset int) ([]*MessageEvent, int, error) {
	query := `SELECT id, master_id, message_id, user_id, event_type, payload, created_at FROM service_messaging_event WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.GetDB().QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var events []*MessageEvent
	for rows.Next() {
		event := &MessageEvent{}
		err := rows.Scan(&event.ID, &event.MasterID, &event.MessageID, &event.UserID, &event.EventType, &event.Payload, &event.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, event)
	}
	// Get total count
	total := 0
	countQuery := `SELECT COUNT(*) FROM service_messaging_event WHERE user_id = $1`
	err = r.GetDB().QueryRowContext(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return events, total, rows.Err()
}

// --- MessagingService RPC-aligned repository methods ---

// SendMessage creates and persists a new direct or thread message.
func (r *Repository) SendMessage(ctx context.Context, req *messagingpb.SendMessageRequest) (*Message, error) {
	// Validate sender/recipients (omitted for brevity)
	masterName := r.GenerateMasterName(repository.EntityType("message"), req.Content, req.Type.String(), fmt.Sprintf("sender-%s", req.SenderId))
	masterID, err := r.masterRepo.Create(ctx, repository.EntityType("message"), masterName)
	if err != nil {
		return nil, err
	}
	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, err = protojson.Marshal(req.Metadata)
		if err != nil {
			return nil, err
		}
	}
	attachments, err := json.Marshal(req.Attachments)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal attachments", zap.Error(err))
		}
	}
	reactions, err := json.Marshal([]interface{}{}) // empty at creation
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal reactions", zap.Error(err))
		}
	}
	msg := &Message{}
	query := `INSERT INTO service_messaging_main (
		master_id, thread_id, conversation_id, chat_group_id, sender_id, recipient_ids, content, type, attachments, reactions, status, edited, deleted, metadata, campaign_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW()
	) RETURNING id, created_at, updated_at`
	err = r.GetDB().QueryRowContext(ctx, query,
		masterID, req.ThreadId, req.ConversationId, req.ChatGroupId, req.SenderId, pq.Array(req.RecipientIds), req.Content, req.Type.String(), attachments, reactions, messagingpb.MessageStatus_MESSAGE_STATUS_SENT.String(), false, false, metadataJSON, req.CampaignId,
	).Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)
	if err != nil {
		if err := r.masterRepo.Delete(ctx, masterID); err != nil {
			if logInstance != nil {
				logInstance.Error("failed to delete master record", zap.Error(err))
			}
		}
		return nil, err
	}
	msg.MasterID = masterID
	msg.ThreadID = req.ThreadId
	msg.ConversationID = req.ConversationId
	msg.ChatGroupID = req.ChatGroupId
	msg.SenderID = req.SenderId
	msg.RecipientIDs = req.RecipientIds
	msg.Content = req.Content
	msg.Type = req.Type.String()
	msg.Attachments = attachments
	msg.Reactions = reactions
	msg.Status = messagingpb.MessageStatus_MESSAGE_STATUS_SENT.String()
	msg.Edited = false
	msg.Deleted = false
	msg.Metadata = req.Metadata
	msg.CampaignID = req.CampaignId
	return msg, nil
}

// SendGroupMessage creates and persists a new group message.
func (r *Repository) SendGroupMessage(ctx context.Context, req *messagingpb.SendGroupMessageRequest) (*Message, error) {
	// Validate group and sender (omitted for brevity)
	masterName := r.GenerateMasterName(repository.EntityType("message"), req.Content, req.Type.String(), fmt.Sprintf("sender-%s", req.SenderId))
	masterID, err := r.masterRepo.Create(ctx, repository.EntityType("message"), masterName)
	if err != nil {
		return nil, err
	}
	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, err = protojson.Marshal(req.Metadata)
		if err != nil {
			return nil, err
		}
	}
	attachments, err := json.Marshal(req.Attachments)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal attachments", zap.Error(err))
		}
	}
	reactions, err := json.Marshal([]interface{}{})
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal reactions", zap.Error(err))
		}
	}
	msg := &Message{}
	query := `INSERT INTO service_messaging_main (
		master_id, chat_group_id, sender_id, content, type, attachments, reactions, status, edited, deleted, metadata, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
	) RETURNING id, created_at, updated_at`
	err = r.GetDB().QueryRowContext(ctx, query,
		masterID, req.ChatGroupId, req.SenderId, req.Content, req.Type.String(), attachments, reactions, messagingpb.MessageStatus_MESSAGE_STATUS_SENT.String(), false, false, metadataJSON,
	).Scan(&msg.ID, &msg.CreatedAt, &msg.UpdatedAt)
	if err != nil {
		if err := r.masterRepo.Delete(ctx, masterID); err != nil {
			if logInstance != nil {
				logInstance.Error("failed to delete master record", zap.Error(err))
			}
		}
		return nil, err
	}
	msg.MasterID = masterID
	msg.ChatGroupID = req.ChatGroupId
	msg.SenderID = req.SenderId
	msg.Content = req.Content
	msg.Type = req.Type.String()
	msg.Attachments = attachments
	msg.Reactions = reactions
	msg.Status = messagingpb.MessageStatus_MESSAGE_STATUS_SENT.String()
	msg.Edited = false
	msg.Deleted = false
	msg.Metadata = req.Metadata
	return msg, nil
}

// EditMessage updates the content/attachments of a message.
func (r *Repository) EditMessage(ctx context.Context, req *messagingpb.EditMessageRequest) (*Message, error) {
	msg, err := r.GetMessage(ctx, req.MessageId)
	if err != nil {
		return nil, err
	}
	msg.Content = req.NewContent
	if req.NewAttachments != nil {
		msg.Attachments, err = json.Marshal(req.NewAttachments)
		if err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to marshal new attachments", zap.Error(err))
			}
		}
	}
	msg.Edited = true
	if msg.Metadata == nil {
		msg.Metadata = &commonpb.Metadata{}
	}
	if msg.Metadata.ServiceSpecific == nil {
		msg.Metadata.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	// Optionally update audit/versioning in metadata
	var metadataJSON []byte
	metadataJSON, err = protojson.Marshal(msg.Metadata)
	if err != nil {
		return nil, err
	}
	query := `UPDATE service_messaging_main SET content=$1, attachments=$2, edited=true, metadata=$3, updated_at=NOW() WHERE id=$4`
	_, err = r.GetDB().ExecContext(ctx, query, msg.Content, msg.Attachments, metadataJSON, msg.ID)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// DeleteMessageByRequest marks a message as deleted or removes it (RPC-aligned).
func (r *Repository) DeleteMessageByRequest(ctx context.Context, req *messagingpb.DeleteMessageRequest) (bool, error) {
	msg, err := r.GetMessage(ctx, req.MessageId)
	if err != nil {
		return false, err
	}
	msg.Deleted = true
	if msg.Metadata == nil {
		msg.Metadata = &commonpb.Metadata{}
	}
	if msg.Metadata.ServiceSpecific == nil {
		msg.Metadata.ServiceSpecific = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	// Optionally update audit/compliance in metadata
	metadataJSON, err := protojson.Marshal(msg.Metadata)
	if err != nil {
		return false, err
	}
	query := `UPDATE service_messaging_main SET deleted=true, metadata=$1, updated_at=NOW() WHERE id=$2`
	_, err = r.GetDB().ExecContext(ctx, query, metadataJSON, msg.ID)
	if err != nil {
		return false, err
	}
	return true, nil
}

// ReactToMessage adds or updates a reaction on a message.
func (r *Repository) ReactToMessage(ctx context.Context, req *messagingpb.ReactToMessageRequest) (*Message, error) {
	msg, err := r.GetMessage(ctx, req.MessageId)
	if err != nil {
		return nil, err
	}
	var reactions []*messagingpb.Reaction
	if msg.Reactions != nil {
		if err := json.Unmarshal(msg.Reactions, &reactions); err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to unmarshal reactions", zap.Error(err))
			}
		}
	}
	reaction := &messagingpb.Reaction{
		UserId:    req.UserId,
		Emoji:     req.Emoji,
		ReactedAt: nil, // Set server time if needed
		Metadata:  req.Metadata,
	}
	// Replace or add reaction
	found := false
	for i := range reactions {
		if reactions[i].UserId != req.UserId {
			continue
		}
		reactions[i].UserId = reaction.UserId
		reactions[i].Emoji = reaction.Emoji
		reactions[i].ReactedAt = reaction.ReactedAt
		reactions[i].Metadata = reaction.Metadata
		found = true
		break
	}
	if !found {
		reactions = append(reactions, reaction)
	}
	msg.Reactions, err = json.Marshal(reactions)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal reactions", zap.Error(err))
		}
		return nil, err
	}
	metadataJSON, err := protojson.Marshal(msg.Metadata)
	if err != nil {
		return nil, err
	}
	query := `UPDATE service_messaging_main SET reactions=$1, metadata=$2, updated_at=NOW() WHERE id=$3`
	_, err = r.GetDB().ExecContext(ctx, query, msg.Reactions, metadataJSON, msg.ID)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetMessage retrieves a message by ID (RPC-aligned).
func (r *Repository) GetMessageByID(ctx context.Context, req *messagingpb.GetMessageRequest) (*Message, error) {
	return r.GetMessage(ctx, req.MessageId)
}

// ListMessages retrieves messages for a thread, conversation, or group (RPC-aligned).
func (r *Repository) ListMessagesByFilter(ctx context.Context, req *messagingpb.ListMessagesRequest) ([]*Message, int, error) {
	// Support filtering by thread, conversation, group, and metadata
	query := `SELECT id, master_id, thread_id, conversation_id, chat_group_id, sender_id, recipient_ids, content, type, attachments, reactions, status, edited, deleted, metadata, campaign_id, created_at, updated_at FROM service_messaging_main WHERE campaign_id = $1 AND (thread_id = $2 OR conversation_id = $3 OR chat_group_id = $4) ORDER BY created_at DESC LIMIT $5 OFFSET $6`
	rows, err := r.GetDB().QueryContext(ctx, query, req.CampaignId, req.ThreadId, req.ConversationId, req.ChatGroupId, req.PageSize, (req.Page-1)*req.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var metadataStr string
		err := rows.Scan(&msg.ID, &msg.MasterID, &msg.ThreadID, &msg.ConversationID, &msg.ChatGroupID, &msg.SenderID, pq.Array(&msg.RecipientIDs), &msg.Content, &msg.Type, &msg.Attachments, &msg.Reactions, &msg.Status, &msg.Edited, &msg.Deleted, &metadataStr, &msg.CampaignID, &msg.CreatedAt, &msg.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		if metadataStr != "" {
			msg.Metadata = &commonpb.Metadata{}
			err := protojson.Unmarshal([]byte(metadataStr), msg.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal message metadata", zap.Error(err))
				return nil, 0, err
			}
		} else {
			msg.Metadata = &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": "metadata unavailable", "message_id": msg.ID}, safeServiceSpecific(msg.Metadata)),
				Tags:            []string{},
				Features:        []string{},
			}
		}
		messages = append(messages, msg)
	}
	// For total count, run a separate count query (omitted for brevity)
	return messages, len(messages), rows.Err()
}

// ListThreads retrieves threads for a user or context (RPC-aligned).
func (r *Repository) ListThreadsByUser(ctx context.Context, req *messagingpb.ListThreadsRequest) ([]*Thread, int, error) {
	query := `SELECT id, master_id, subject, participant_ids, message_ids, metadata, campaign_id, created_at, updated_at FROM service_messaging_thread WHERE $1 = ANY(participant_ids) AND campaign_id = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.GetDB().QueryContext(ctx, query, req.UserId, req.CampaignId, req.PageSize, (req.Page-1)*req.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var threads []*Thread
	for rows.Next() {
		thread := &Thread{}
		var metadataStr string
		err := rows.Scan(&thread.ID, &thread.MasterID, &thread.Subject, pq.Array(&thread.ParticipantIDs), pq.Array(&thread.MessageIDs), &metadataStr, &thread.CampaignID, &thread.CreatedAt, &thread.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		thread.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), thread.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal thread metadata", zap.Error(err))
				return nil, 0, err
			}
		}
		threads = append(threads, thread)
	}
	return threads, len(threads), rows.Err()
}

// ListConversations retrieves conversations for a user or context (RPC-aligned).
func (r *Repository) ListConversationsByUser(ctx context.Context, req *messagingpb.ListConversationsRequest) ([]*Conversation, int, error) {
	query := `SELECT id, master_id, participant_ids, chat_group_id, thread_ids, metadata, campaign_id, created_at, updated_at FROM service_messaging_conversation WHERE $1 = ANY(participant_ids) ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.GetDB().QueryContext(ctx, query, req.UserId, req.PageSize, (req.Page-1)*req.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var conversations []*Conversation
	for rows.Next() {
		conv := &Conversation{}
		var metadataStr string
		err := rows.Scan(&conv.ID, &conv.MasterID, pq.Array(&conv.ParticipantIDs), &conv.ChatGroupID, pq.Array(&conv.ThreadIDs), &metadataStr, &conv.CampaignID, &conv.CreatedAt, &conv.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		conv.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), conv.Metadata)
			if err != nil {
				logInstance.Warn("failed to unmarshal conversation metadata", zap.Error(err))
				return nil, 0, err
			}
		}
		conversations = append(conversations, conv)
	}
	return conversations, len(conversations), rows.Err()
}

// MarkAsRead marks a message as read for a user.
func (r *Repository) MarkAsRead(ctx context.Context, req *messagingpb.MarkAsReadRequest) (bool, error) {
	// Log event and update metadata (audit, context)
	// Insert event
	_, err := r.GetDB().ExecContext(ctx,
		`INSERT INTO service_messaging_event (message_id, user_id, event_type, payload, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		req.MessageId, req.UserId, "read", "{}")
	if err != nil {
		return false, err
	}
	// Optionally update message metadata (not implemented here)
	return true, nil
}

// MarkAsDelivered marks a message as delivered for a user.
func (r *Repository) MarkAsDelivered(ctx context.Context, req *messagingpb.MarkAsDeliveredRequest) (bool, error) {
	// Log event and update metadata
	_, err := r.GetDB().ExecContext(ctx,
		`INSERT INTO service_messaging_event (message_id, user_id, event_type, payload, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		req.MessageId, req.UserId, "delivered", "{}")
	if err != nil {
		return false, err
	}
	return true, nil
}

// AcknowledgeMessage acknowledges a message for a user.
func (r *Repository) AcknowledgeMessage(ctx context.Context, req *messagingpb.AcknowledgeMessageRequest) (bool, error) {
	// Log event and update metadata
	_, err := r.GetDB().ExecContext(ctx,
		`INSERT INTO service_messaging_event (message_id, user_id, event_type, payload, created_at) VALUES ($1, $2, $3, $4, NOW())`,
		req.MessageId, req.UserId, "acknowledged", "{}")
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreateChatGroup creates a new chat group.
func (r *Repository) CreateChatGroupWithRequest(ctx context.Context, req *messagingpb.CreateChatGroupRequest) (*ChatGroup, error) {
	masterName := r.GenerateMasterName(repository.EntityType("chat_group"), req.Name, "group", "")
	masterID, err := r.masterRepo.Create(ctx, repository.EntityType("chat_group"), masterName)
	if err != nil {
		return nil, err
	}
	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, err = protojson.Marshal(req.Metadata)
		if err != nil {
			return nil, err
		}
	}
	roles, err := json.Marshal(req.Roles)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal roles", zap.Error(err))
		}
	}
	group := &ChatGroup{}
	query := `INSERT INTO service_messaging_chat_group (
		master_id, name, description, member_ids, roles, metadata, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, NOW(), NOW()
	) RETURNING id, created_at, updated_at`
	err = r.GetDB().QueryRowContext(ctx, query,
		masterID, req.Name, req.Description, pq.Array(req.MemberIds), roles, metadataJSON,
	).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		if err := r.masterRepo.Delete(ctx, masterID); err != nil {
			if logInstance != nil {
				logInstance.Error("failed to delete master record", zap.Error(err))
			}
		}
		return nil, err
	}
	group.MasterID = masterID
	group.Name = req.Name
	group.Description = req.Description
	group.MemberIDs = req.MemberIds
	group.Roles = req.Roles
	group.Metadata = req.Metadata
	group.CampaignID = req.CampaignId
	return group, nil
}

// AddChatGroupMember adds a member to a chat group.
func (r *Repository) AddChatGroupMember(ctx context.Context, req *messagingpb.AddChatGroupMemberRequest) (*ChatGroup, error) {
	// Fetch group
	group := &ChatGroup{}
	var metadataStr, rolesStr string
	query := `SELECT id, master_id, name, description, member_ids, roles, metadata, created_at, updated_at FROM service_messaging_chat_group WHERE id = $1`
	err := r.GetDB().QueryRowContext(ctx, query, req.ChatGroupId).Scan(&group.ID, &group.MasterID, &group.Name, &group.Description, pq.Array(&group.MemberIDs), &rolesStr, &metadataStr, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return nil, err
	}
	group.Metadata = &commonpb.Metadata{}
	if metadataStr != "" {
		if err := protojson.Unmarshal([]byte(metadataStr), group.Metadata); err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to unmarshal metadata", zap.Error(err))
			}
		}
	}
	group.Roles = map[string]string{}
	if rolesStr != "" {
		if err := json.Unmarshal([]byte(rolesStr), &group.Roles); err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to unmarshal roles", zap.Error(err))
			}
		}
	}
	// Add member if not present
	found := false
	for _, id := range group.MemberIDs {
		if id == req.UserId {
			found = true
			break
		}
	}
	if !found {
		group.MemberIDs = append(group.MemberIDs, req.UserId)
	}
	if req.Role != "" {
		group.Roles[req.UserId] = req.Role
	}
	// Optionally update metadata
	metadataJSON, err := protojson.Marshal(group.Metadata)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal metadata", zap.Error(err))
		}
	}
	rolesJSON, err := json.Marshal(group.Roles)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal roles", zap.Error(err))
		}
	}
	updateQuery := `UPDATE service_messaging_chat_group SET member_ids=$1, roles=$2, metadata=$3, updated_at=NOW() WHERE id=$4`
	_, err = r.GetDB().ExecContext(ctx, updateQuery, pq.Array(group.MemberIDs), rolesJSON, metadataJSON, group.ID)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// RemoveChatGroupMember removes a member from a chat group.
func (r *Repository) RemoveChatGroupMember(ctx context.Context, req *messagingpb.RemoveChatGroupMemberRequest) (*ChatGroup, error) {
	// Fetch group
	group := &ChatGroup{}
	var metadataStr, rolesStr string
	query := `SELECT id, master_id, name, description, member_ids, roles, metadata, created_at, updated_at FROM service_messaging_chat_group WHERE id = $1`
	err := r.GetDB().QueryRowContext(ctx, query, req.ChatGroupId).Scan(&group.ID, &group.MasterID, &group.Name, &group.Description, pq.Array(&group.MemberIDs), &rolesStr, &metadataStr, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return nil, err
	}
	group.Metadata = &commonpb.Metadata{}
	if metadataStr != "" {
		if err := protojson.Unmarshal([]byte(metadataStr), group.Metadata); err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to unmarshal metadata", zap.Error(err))
			}
		}
	}
	group.Roles = map[string]string{}
	if rolesStr != "" {
		if err := json.Unmarshal([]byte(rolesStr), &group.Roles); err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to unmarshal roles", zap.Error(err))
			}
		}
	}
	// Remove member
	newMembers := make([]string, 0, len(group.MemberIDs))
	for _, id := range group.MemberIDs {
		if id != req.UserId {
			newMembers = append(newMembers, id)
		}
	}
	group.MemberIDs = newMembers
	delete(group.Roles, req.UserId)
	// Optionally update metadata
	metadataJSON, err := protojson.Marshal(group.Metadata)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal metadata", zap.Error(err))
		}
	}
	rolesJSON, err := json.Marshal(group.Roles)
	if err != nil {
		if logInstance != nil {
			logInstance.Warn("failed to marshal roles", zap.Error(err))
		}
	}
	updateQuery := `UPDATE service_messaging_chat_group SET member_ids=$1, roles=$2, metadata=$3, updated_at=NOW() WHERE id=$4`
	_, err = r.GetDB().ExecContext(ctx, updateQuery, pq.Array(group.MemberIDs), rolesJSON, metadataJSON, group.ID)
	if err != nil {
		return nil, err
	}
	return group, nil
}

// GetChatGroupByID fetches a chat group by its ID.
func (r *Repository) GetChatGroupByID(ctx context.Context, id string) (*ChatGroup, error) {
	group := &ChatGroup{}
	var metadataStr, rolesStr string
	query := `SELECT id, master_id, name, description, member_ids, roles, metadata, created_at, updated_at FROM service_messaging_chat_group WHERE id = $1`
	err := r.GetDB().QueryRowContext(ctx, query, id).Scan(&group.ID, &group.MasterID, &group.Name, &group.Description, pq.Array(&group.MemberIDs), &rolesStr, &metadataStr, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrChatGroupNotFound
		}
		return nil, err
	}
	group.Metadata = &commonpb.Metadata{}
	if metadataStr != "" {
		if err := protojson.Unmarshal([]byte(metadataStr), group.Metadata); err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to unmarshal chat group metadata", zap.Error(err))
			}
		}
	}
	group.Roles = map[string]string{}
	if rolesStr != "" {
		if err := json.Unmarshal([]byte(rolesStr), &group.Roles); err != nil {
			if logInstance != nil {
				logInstance.Warn("failed to unmarshal roles", zap.Error(err))
			}
		}
	}
	return group, nil
}

// UpdateMessagingPreferences upserts preferences for a user.
func (r *Repository) UpdateMessagingPreferences(ctx context.Context, userID string, prefs *messagingpb.MessagingPreferences) error {
	prefsJSON, err := protojson.Marshal(prefs)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO service_messaging_preferences (user_id, preferences, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET preferences = $2, updated_at = NOW()
	`
	_, err = r.GetDB().ExecContext(ctx, query, userID, prefsJSON)
	return err
}

// GetMessagingPreferences fetches preferences for a user.
func (r *Repository) GetMessagingPreferences(ctx context.Context, userID string) (*messagingpb.MessagingPreferences, int64, error) {
	var prefsJSON []byte
	var updatedAt time.Time
	query := `SELECT preferences, updated_at FROM service_messaging_preferences WHERE user_id = $1`
	err := r.GetDB().QueryRowContext(ctx, query, userID).Scan(&prefsJSON, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, nil // Not found, return nil
		}
		return nil, 0, err
	}
	prefs := &messagingpb.MessagingPreferences{}
	if err := protojson.Unmarshal(prefsJSON, prefs); err != nil {
		return nil, 0, err
	}
	return prefs, updatedAt.Unix(), nil
}
