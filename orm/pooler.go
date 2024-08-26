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
	Choose(c context.Context, dbs *DBs) *gorm.DB
}

type RandomPolicy struct {
}

func (*RandomPolicy) Choose(c context.Context, dbs *DBs) *gorm.DB {
	l := len(dbs.dbs)
	if l == 1 {
		return dbs.dbs[0].Orm()
	}
	return dbs.dbs[rand.Intn(l)].Orm()
}
