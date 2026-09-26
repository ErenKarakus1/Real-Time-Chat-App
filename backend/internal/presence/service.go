package presence

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const onlineUsersKey = "presence:online_users"

type Service struct {
	client *redis.Client
	mu     sync.Mutex
	counts map[uuid.UUID]int
}

func NewService(client *redis.Client) *Service {
	return &Service{
		client: client,
		counts: make(map[uuid.UUID]int),
	}
}

func (s *Service) Connect(ctx context.Context, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counts[userID]++
	if s.counts[userID] > 1 {
		return nil
	}

	return s.client.SAdd(ctx, onlineUsersKey, userID.String()).Err()
}

func (s *Service) Disconnect(ctx context.Context, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.counts[userID] <= 1 {
		delete(s.counts, userID)
		return s.client.SRem(ctx, onlineUsersKey, userID.String()).Err()
	}

	s.counts[userID]--
	return nil
}

func (s *Service) Statuses(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	statuses := make(map[uuid.UUID]bool, len(userIDs))
	if len(userIDs) == 0 {
		return statuses, nil
	}

	members := make([]interface{}, 0, len(userIDs))
	for _, userID := range userIDs {
		members = append(members, userID.String())
	}

	results, err := s.client.SMIsMember(ctx, onlineUsersKey, members...).Result()
	if err != nil {
		return nil, err
	}

	for index, userID := range userIDs {
		statuses[userID] = results[index]
	}

	return statuses, nil
}
