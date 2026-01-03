package dataflow

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessAfterRetries", func(t *testing.T) {
		var attempts int32
		fn := func(msg interface{}) (interface{}, error) {
			curr := atomic.AddInt32(&attempts, 1)
			if curr < 3 {
				return nil, errors.New("fail")
			}
			return "success", nil
		}

		src := From(ctx, "item1")
		res := Map(ctx, src, fn, WithRetry(3, ConstantBackoff(10*time.Millisecond)))

		var results []interface{}
		err := ForEach(ctx, res, func(msg interface{}) error {
			results = append(results, msg)
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(results))
		assert.Equal(t, "success", results[0])
		assert.Equal(t, int32(3), atomic.LoadInt32(&attempts))
	})

	t.Run("FailAfterMaxRetries", func(t *testing.T) {
		var attempts int32
		fn := func(msg interface{}) (interface{}, error) {
			atomic.AddInt32(&attempts, 1)
			return nil, errors.New("permanent fail")
		}

		src := From(ctx, "item1")
		// errorHandler returns false to allow the item to be dropped (default behavior in Map if not handled)
		// but here we want to see if it actually retries 3 times (total 4 attempts: 0, 1, 2, 3)
		res := Map(ctx, src, fn, WithRetry(3, ConstantBackoff(1*time.Millisecond)))

		var results []interface{}
		err := ForEach(ctx, res, func(msg interface{}) error {
			results = append(results, msg)
			return nil
		})

		assert.NoError(t, err) // ForEach doesn't fail because Map drops the item after retries exceed
		assert.Equal(t, 0, len(results))
		assert.Equal(t, int32(4), atomic.LoadInt32(&attempts))
	})

	t.Run("ExponentialBackoff", func(t *testing.T) {
		backoff := ExponentialBackoff(10 * time.Millisecond)
		assert.Equal(t, 10*time.Millisecond, backoff(0))
		assert.Equal(t, 10*time.Millisecond, backoff(1))
		assert.Equal(t, 20*time.Millisecond, backoff(2))
		assert.Equal(t, 40*time.Millisecond, backoff(3))
		assert.Equal(t, 80*time.Millisecond, backoff(4))
	})
}
