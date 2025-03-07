package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/0glabs/0g-data-avail/common"
	"github.com/0glabs/0g-data-avail/common/mock"
	"github.com/0glabs/0g-data-avail/common/ratelimit"
	"github.com/0glabs/0g-data-avail/common/store"
	"github.com/stretchr/testify/assert"
)

func makeTestRatelimiter() (common.RateLimiter, error) {

	globalParams := common.GlobalRateParams{
		BucketSizes: []time.Duration{time.Second, time.Minute},
		Multipliers: []float32{1, 1},
	}
	bucketStoreSize := 1000

	bucketStore, err := store.NewLocalParamStore[common.RateBucketParams](bucketStoreSize)
	if err != nil {
		return nil, err
	}

	ratelimiter := ratelimit.NewRateLimiter(globalParams, bucketStore, []string{"testRetriever2"}, &mock.Logger{})

	return ratelimiter, nil

}

func TestRatelimit(t *testing.T) {

	ratelimiter, err := makeTestRatelimiter()
	assert.NoError(t, err)

	ctx := context.Background()

	retreiverID := "testRetriever"

	for i := 0; i < 10; i++ {
		allow, err := ratelimiter.AllowRequest(ctx, retreiverID, 10, 100)
		assert.NoError(t, err)
		assert.Equal(t, true, allow)
	}

	allow, err := ratelimiter.AllowRequest(ctx, retreiverID, 10, 100)
	assert.NoError(t, err)
	assert.Equal(t, false, allow)
}

func TestRatelimitAllowlist(t *testing.T) {
	ratelimiter, err := makeTestRatelimiter()
	assert.NoError(t, err)

	ctx := context.Background()

	retreiverID := "testRetriever2"

	// 10x more requests allowed for allowlisted IDs
	for i := 0; i < 100; i++ {
		allow, err := ratelimiter.AllowRequest(ctx, retreiverID, 10, 100)
		assert.NoError(t, err)
		assert.Equal(t, true, allow)
	}
}
