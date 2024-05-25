package orm

/*
 * @abstract Orm client
 * @mail neo532@126.com
 * @date 2024-05-18
 */

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type contextTransactionKey struct{}

type Orms struct {
	read        []*gorm.DB
	write       []*gorm.DB
	shadowRead  []*gorm.DB
	shadowWrite []*gorm.DB

	pooler      Pooler
	benchmarker Benchmarker

	cleanupFuncs []func()
	err          error
}

// ========== OrmsOpt =========
type OrmsOpt func(*Orms)

func WithBenchmarker(fn Benchmarker) OrmsOpt {
	return func(o *Orms) {
		o.benchmarker = fn
	}
}

func WithPooler(fn Pooler) OrmsOpt {
	return func(o *Orms) {
		o.pooler = fn
	}
}

func WithRead(reads ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if len(reads) == 0 {
			return
		}
		for _, db := range reads {
			o.read = append(o.read, o.setDB(db))
		}
	}
}

func WithWrite(writes ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if len(writes) == 0 {
			return
		}
		for _, db := range writes {
			o.write = append(o.write, o.setDB(db))
		}
	}
}

func WithShadowRead(reads ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if len(reads) == 0 {
			return
		}
		for _, db := range reads {
			o.shadowRead = append(o.shadowRead, o.setDB(db))
		}
	}
}

func WithShadowWrite(writes ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if len(writes) == 0 {
			return
		}
		for _, db := range writes {
			o.shadowWrite = append(o.shadowWrite, o.setDB(db))
		}
	}
}

// ========== /OrmsOpt =========

func News(opts ...OrmsOpt) (dbs *Orms) {
	dbs = &Orms{
		benchmarker: &DefaultBenchmarker{},
		pooler:      &RandomPolicy{},
	}
	for _, opt := range opts {
		opt(dbs)
	}
	if dbs.write == nil && dbs.read == nil {
		dbs.err = errors.New("Please input a instance at least")
	}
	return
}

func (m *Orms) Read(c context.Context) (db *gorm.DB) {
	if tx, ok := c.Value(contextTransactionKey{}).(*gorm.DB); ok {
		return tx
	}
	if m.benchmarker.Judge(c) {
		if m.shadowRead == nil {
			return m.pooler.Choose(c, m.shadowWrite).WithContext(c)
		}
		return m.pooler.Choose(c, m.shadowRead).WithContext(c)
	}
	if m.read == nil {
		return m.pooler.Choose(c, m.write).WithContext(c)
	}
	return m.pooler.Choose(c, m.read).WithContext(c)
}

func (m *Orms) Write(c context.Context) (db *gorm.DB) {
	if tx, ok := c.Value(contextTransactionKey{}).(*gorm.DB); ok {
		return tx
	}
	if m.benchmarker.Judge(c) {
		if m.shadowWrite == nil {
			return m.pooler.Choose(c, m.shadowRead).WithContext(c)
		}
		return m.pooler.Choose(c, m.shadowWrite).WithContext(c)
	}
	if m.write == nil {
		return m.pooler.Choose(c, m.read).WithContext(c)
	}
	return m.pooler.Choose(c, m.write).WithContext(c)
}

func (m *Orms) Transaction(c context.Context, fn func(c context.Context) error) error {
	return m.Write(c).Transaction(func(tx *gorm.DB) error {
		c = context.WithValue(c, contextTransactionKey{}, tx)
		return fn(c)
	})
}

func (m *Orms) Cleanup() func() {
	return func() {
		for _, fn := range m.cleanupFuncs {
			fn()
		}
	}
}

func (m *Orms) setDB(db *Orm) *gorm.DB {
	m.cleanupFuncs = append(m.cleanupFuncs, db.Cleanup())
	if err := db.Error(); err != nil {
		m.err = err
	}
	return db.Orm()
}

func (m *Orms) Error() error {
	return m.err
}
