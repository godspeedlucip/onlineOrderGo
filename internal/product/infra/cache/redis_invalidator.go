package cache

import (
	"context"
	"fmt"
)

type RedisInvalidator struct {
	readCache *RedisReadCache
}

func NewRedisInvalidator(readCache *RedisReadCache) *RedisInvalidator {
	return &RedisInvalidator{readCache: readCache}
}

func (i *RedisInvalidator) InvalidateCategory(ctx context.Context, categoryID int64) error {
	_ = ctx
	if i.readCache == nil {
		return nil
	}
	if err := i.readCache.deleteByPrefix("product:category:"); err != nil {
		return err
	}
	if err := i.readCache.deleteByPrefix(fmt.Sprintf("product:dish:cid=%d:", categoryID)); err != nil {
		return err
	}
	if err := i.readCache.deleteByPrefix(fmt.Sprintf("product:setmeal:cid=%d:", categoryID)); err != nil {
		return err
	}
	return nil
}

func (i *RedisInvalidator) InvalidateDish(ctx context.Context, dishID int64, categoryID int64) error {
	_ = ctx
	_ = dishID
	if i.readCache == nil {
		return nil
	}
	if err := i.readCache.deleteByPrefix("product:dish:"); err != nil {
		return err
	}
	if categoryID > 0 {
		if err := i.readCache.deleteByPrefix(fmt.Sprintf("product:dish:cid=%d:", categoryID)); err != nil {
			return err
		}
	}
	return nil
}

func (i *RedisInvalidator) InvalidateSetmeal(ctx context.Context, setmealID int64, categoryID int64) error {
	_ = ctx
	_ = setmealID
	if i.readCache == nil {
		return nil
	}
	if err := i.readCache.deleteByPrefix("product:setmeal:"); err != nil {
		return err
	}
	if categoryID > 0 {
		if err := i.readCache.deleteByPrefix(fmt.Sprintf("product:setmeal:cid=%d:", categoryID)); err != nil {
			return err
		}
	}
	return nil
}

func (i *RedisInvalidator) InvalidateByCategory(ctx context.Context, categoryID int64) error {
	_ = ctx
	if i.readCache == nil {
		return nil
	}
	if err := i.readCache.deleteByPrefix("product:category:"); err != nil {
		return err
	}
	if err := i.readCache.deleteByPrefix(fmt.Sprintf("product:dish:cid=%d:", categoryID)); err != nil {
		return err
	}
	if err := i.readCache.deleteByPrefix(fmt.Sprintf("product:setmeal:cid=%d:", categoryID)); err != nil {
		return err
	}
	return nil
}

// TODO: if Java side has fine-grained key patterns, align invalidation scope exactly.
