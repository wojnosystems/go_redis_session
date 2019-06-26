//Copyright 2019 Chris Wojno
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
// documentation files (the "Software"), to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the
// Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
// WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS
// OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR
// OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package go_redis_session

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/wojnosystems/go_session_store"
	"time"
)

// NewRedisStore creates a new session store, powered by a Redis server
// @param client is the redis.Client object, configured and ready for use
// @param sessionDuration is how long sessions will be valid for before Redis purges them
// @param builder is how new session identifiers should be created. Use NewRandomSource or your own if you like
func NewRedisClusterStore(client *redis.ClusterClient, sessionDuration time.Duration, builder go_session_store.SessionIdGenerator) go_session_store.SessionStorer {
	return &redisClusterStore{
		redisClient:     client,
		sessionDuration: sessionDuration,
		builder:         builder,
	}
}

type redisClusterStore struct {
	redisClient     *redis.ClusterClient
	sessionDuration time.Duration
	builder         go_session_store.SessionIdGenerator
}

func (r *redisClusterStore) GenerateAndStore(ctx context.Context, userId string, metaData string) (session []byte, err error) {
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

	return session, redisClusterCheckAndSet(r.redisClient, redisKeyFromSession(session), string(sdJson), r.sessionDuration)
}

// Ensure that the key is not already in use and was not taken while we were checking (Check and Set)
func redisClusterCheckAndSet(client *redis.ClusterClient, key string, values string, sessionDuration time.Duration) (err error) {
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

func (r *redisClusterStore) Get(ctx context.Context, session []byte) (userId string, metaData string, err error) {
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
