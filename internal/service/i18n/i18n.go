package i18n

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	i18npb "github.com/nmxmxh/master-ovasabi/api/protos/i18n/v1"
	i18nrepo "github.com/nmxmxh/master-ovasabi/internal/repository/i18n"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ServiceImpl implements the I18nService interface.
type ServiceImpl struct {
	i18npb.UnimplementedI18NServiceServer
	log              *zap.Logger
	cache            *redis.Cache
	repo             *i18nrepo.Repository
	supportedLocales []string
	defaultLocale    string
}

// NewService creates a new instance of I18nService.
func NewService(log *zap.Logger, repo *i18nrepo.Repository, cache *redis.Cache) *ServiceImpl {
	return &ServiceImpl{
		log:              log,
		repo:             repo,
		cache:            cache,
		supportedLocales: []string{"en", "es", "fr", "de", "it", "pt", "ru", "zh", "ja", "ko", "ar", "tr", "pl", "nl", "sv", "fi", "no", "da", "cs", "el", "he", "hi", "id", "ms", "ro", "th", "uk", "vi", "bg", "hr", "hu", "sk", "sl", "sr", "ca", "et", "fa", "lt", "lv", "mt", "tl", "bn", "sw", "ta", "te", "ur", "ml", "gu", "kn", "mr", "pa", "si", "my", "km", "lo", "am", "zu", "xh", "st", "tn", "ts", "ss", "ve", "nr", "af", "eu", "gl", "is", "ga", "cy", "lb", "mk", "sq", "az", "be", "ka", "hy", "kk", "ky", "mn", "tg", "tk", "uz", "tt", "ba", "cv", "os", "mo", "ab", "av", "ce", "cu", "kv", "udm", "kom", "sah", "ch", "jv", "su", "ace", "ban", "bug", "mad", "min", "ms", "id", "tet", "fil", "ilo", "pam", "war", "yue"},
		defaultLocale:    "en",
	}
}

func (s *ServiceImpl) CreateTranslation(ctx context.Context, req *i18npb.CreateTranslationRequest) (*i18npb.CreateTranslationResponse, error) {
	description := ""
	tags := ""
	if req.Metadata != nil {
		if d, ok := req.Metadata["description"]; ok {
			description = d
		}
		if t, ok := req.Metadata["tags"]; ok {
			tags = t
		}
	}

	translation := &i18nrepo.Translation{
		MasterID:    int64(req.MasterId),
		Key:         req.Key,
		Locale:      req.Language,
		Value:       req.Value,
		Description: description,
		Tags:        tags,
	}
	created, err := s.repo.Create(ctx, translation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create translation: %v", err)
	}

	// Cache in Redis
	cacheKey := fmt.Sprintf("i18n:%s:%s", created.Key, created.Locale)
	if err := s.cache.Set(ctx, cacheKey, "", created.Value, 0); err != nil {
		s.log.Warn("failed to cache translation", zap.String("key", cacheKey), zap.Error(err))
	}

	metadata := map[string]string{}
	if created.Description != "" {
		metadata["description"] = created.Description
	}
	if created.Tags != "" {
		metadata["tags"] = created.Tags
	}

	return &i18npb.CreateTranslationResponse{
		Translation: &i18npb.Translation{
			Id:        int32(created.ID),
			MasterId:  int32(created.MasterID),
			Key:       created.Key,
			Language:  created.Locale,
			Value:     created.Value,
			Metadata:  metadata,
			CreatedAt: timestamppb.New(created.CreatedAt),
		},
		Success: true,
	}, nil
}

func (s *ServiceImpl) GetTranslation(ctx context.Context, req *i18npb.GetTranslationRequest) (*i18npb.GetTranslationResponse, error) {
	translation, err := s.repo.GetByID(ctx, int64(req.TranslationId))
	if err != nil {
		if err == i18nrepo.ErrTranslationNotFound {
			return nil, status.Error(codes.NotFound, "translation not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get translation: %v", err)
	}

	return &i18npb.GetTranslationResponse{
		Translation: &i18npb.Translation{
			Id:        int32(translation.ID),
			MasterId:  int32(translation.MasterID),
			Key:       translation.Key,
			Language:  translation.Locale,
			Value:     translation.Value,
			CreatedAt: timestamppb.New(translation.CreatedAt),
		},
	}, nil
}

func (s *ServiceImpl) ListTranslations(ctx context.Context, req *i18npb.ListTranslationsRequest) (*i18npb.ListTranslationsResponse, error) {
	limit := int(req.PageSize)
	if limit <= 0 {
		limit = 20
	}
	offset := int((req.Page - 1) * req.PageSize)
	translations, err := s.repo.ListByLocale(ctx, req.Language, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list translations: %v", err)
	}

	resp := &i18npb.ListTranslationsResponse{
		Translations: make([]*i18npb.Translation, 0, len(translations)),
		Page:         req.Page,
		TotalCount:   int32(len(translations)), // Optionally, use a count query for accuracy
		TotalPages:   1,                        // Optionally, calculate based on total count
	}
	for _, t := range translations {
		resp.Translations = append(resp.Translations, &i18npb.Translation{
			Id:        int32(t.ID),
			MasterId:  int32(t.MasterID),
			Key:       t.Key,
			Language:  t.Locale,
			Value:     t.Value,
			CreatedAt: timestamppb.New(t.CreatedAt),
		})
	}
	return resp, nil
}

// BatchTranslateLibre calls the LibreTranslate API to translate a batch of texts.
func BatchTranslateLibre(texts []string, sourceLang, targetLang, endpoint string) ([]string, error) {
	payload := map[string]interface{}{
		"q":      texts,
		"source": sourceLang,
		"target": targetLang,
		"format": "text",
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(endpoint+"/translate", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			// Log the error
			zap.L().Warn("failed to close response body", zap.Error(cerr))
		}
	}()

	var result []struct {
		TranslatedText string `json:"translatedText"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	translations := make([]string, len(result))
	for i, r := range result {
		translations[i] = r.TranslatedText
	}
	return translations, nil
}

// TranslateSite translates a batch of site texts to a target language using LibreTranslate.
func (s *ServiceImpl) TranslateSite(ctx context.Context, req *i18npb.TranslateSiteRequest) (*i18npb.TranslateSiteResponse, error) {
	endpoint := os.Getenv("TRANSLATION_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://libretranslate:5000"
	}
	translations, err := BatchTranslateLibre(req.Texts, req.SourceLang, req.TargetLang, endpoint)
	if err != nil {
		return nil, err
	}
	return &i18npb.TranslateSiteResponse{Translations: translations}, nil
}
