package orm

/*
 * @abstract Orm pooler
 * @mail neo532@126.com
 * @date 2024-05-18
 */

import (
	"context"
	"math/rand"

	"gorm.io/gorm"
)

type Pooler interface {
	Choose(c context.Context, dbs []*gorm.DB) *gorm.DB
}

type RandomPolicy struct {
}

func (*RandomPolicy) Choose(c context.Context, dbs []*gorm.DB) *gorm.DB {
	l := len(dbs)
	if l == 1 {
		return dbs[0]
	}
	return dbs[rand.Intn(l)]
}
