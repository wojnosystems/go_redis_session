package go_redis_session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/wojnosystems/go_session_store"
	"time"
)

// NewRedisStore creates a new session store, powered by a Redis server
// @param client is the redis.Client object, configured and ready for use
// @param sessionDuration is how long sessions will be valid for before Redis purges them
// @param builder is how new session identifiers should be created. Use NewRandomSource or your own if you like
func NewRedisStore(client *redis.Client, sessionDuration time.Duration, builder go_session_store.SessionIdGenerator) go_session_store.SessionStorer {
	return &redisStore{
		redisClient:     client,
		sessionDuration: sessionDuration,
		builder:         builder,
	}
}

type redisStore struct {
	redisClient     *redis.Client
	sessionDuration time.Duration
	builder         go_session_store.SessionIdGenerator
}

func (r *redisStore) GenerateAndStore(ctx context.Context, userId string, metaData string) (session []byte, err error) {
	// Serialize the userId and optional metadata
	sd := redisSessionData{
		UserId:   userId,
		MetaData: metaData,
	}
	sdJson, err := json.Marshal(sd)
	if err != nil {
		return
	}

	// Generate a new session Id
	session, err = r.builder.Generate()
	if err != nil {
		return
	}

	return session, redisCheckAndSet(r.redisClient, redisKeyFromSession(session), string(sdJson), r.sessionDuration)
}

// Ensure that the key is not already in use and was not taken while we were checking (Check and Set)
func redisCheckAndSet(client *redis.Client, key string, values string, sessionDuration time.Duration) (err error) {
	return client.Watch(func(tx *redis.Tx) error {
		result := tx.Get(key)
		if result.Err() != redis.Nil {
			if result.Err() == nil {
				return go_session_store.ErrSessionCollision
			}
			return result.Err()
		}
		_, redErr := tx.Pipelined(func(pipeliner redis.Pipeliner) error {
			pipeliner.Set(key, values, sessionDuration)
			return nil
		})
		if redErr == redis.TxFailedErr {
			return go_session_store.ErrSessionCollision
		}
		return redErr
	}, key)
}

func (r *redisStore) Get(ctx context.Context, session []byte) (userId string, metaData string, err error) {
	status := r.redisClient.Get(redisKeyFromSession(session))
	if status.Err() == redis.Nil {
		return "", "", nil
	} else if status.Err() != nil {
		return "", "", status.Err()
	}

	var sdJson string
	err = status.Scan(&sdJson)
	if err != nil {
		return "", "", err
	}

	sd := redisSessionData{}
	err = json.Unmarshal([]byte(sdJson), &sd)
	return sd.UserId, sd.MetaData, err
}

type redisSessionData struct {
	UserId   string `json:"userId"`
	MetaData string `json:"metaData"`
}

func redisKeyFromSession(sessionId []byte) (key string) {
	return "session-" + base64.StdEncoding.EncodeToString(sessionId)
}
