package go_redis_session

import (
	"context"
	"github.com/go-redis/redis"
	"github.com/wojnosystems/go_session_store"
	"testing"
	"time"
)

func TestRedisStore_GenerateAndStore_Duplicate(t *testing.T) {
	cli := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	cli.FlushAll()
	store := NewRedisStore(cli, 1*time.Hour, &constBuilder{})
	ctx := context.Background()
	_, err := store.GenerateAndStore(ctx, "me", "")
	if err != nil {
		t.Fatal("error not expected")
	}
	// try to store again
	_, err = store.GenerateAndStore(ctx, "me", "")
	if err != go_session_store.ErrSessionCollision {
		t.Error("expected a collision, but did not detect it")
	}
}

func TestRedisStore_GenerateAndStore_OK(t *testing.T) {
	cli := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	cli.FlushAll()
	store := NewRedisStore(cli, 1*time.Hour, &constBuilder{})
	ctx := context.Background()
	session, err := store.GenerateAndStore(ctx, "me", "meta")
	if err != nil {
		t.Fatal("error not expected")
	}
	userId, metaData, err := store.Get(ctx, session)
	if err != nil {
		t.Error("no error expected")
	}
	if userId != "me" {
		t.Error("userId was not expected value")
	}
	if metaData != "meta" {
		t.Error("metadata was not expected value")
	}

}

type constBuilder struct {
}

func (c constBuilder) Generate() ([]byte, error) {
	return []byte("test"), nil
}
