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
