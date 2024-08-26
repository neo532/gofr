package orm

/*
 * @abstract Orm client
 * @mail neo532@126.com
 * @date 2024-05-18
 */

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

type contextTransactionKey struct{}

type Orms struct {
	read        *DBs
	write       *DBs
	shadowRead  *DBs
	shadowWrite *DBs

	pooler      Pooler
	benchmarker Benchmarker

	err error
}

type DBs struct {
	dbs  []*Orm
	lock sync.RWMutex
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

func WithRead(dbs ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if o.read == nil {
			o.read = &DBs{}
		}
		setDB(o.read, o, dbs...)
	}
}

func WithWrite(dbs ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if o.write == nil {
			o.write = &DBs{}
		}
		setDB(o.write, o, dbs...)
	}
}

func WithShadowRead(dbs ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if o.shadowRead == nil {
			o.shadowRead = &DBs{}
		}
		setDB(o.shadowRead, o, dbs...)
	}
}

func WithShadowWrite(dbs ...*Orm) OrmsOpt {
	return func(o *Orms) {
		if o.shadowWrite == nil {
			o.shadowWrite = &DBs{}
		}
		setDB(o.shadowWrite, o, dbs...)
	}
}

func setDB(rs *DBs, o *Orms, dbs ...*Orm) {

	rs.lock.Lock()
	defer rs.lock.Unlock()

	dbOldM := make(map[string]*Orm, len(rs.dbs))
	for _, v := range rs.dbs {
		dbOldM[v.Key()] = v
	}

	dbNew := make([]*Orm, 0, len(dbs))

	var ok bool
	for _, db := range dbs {

		if err := db.Error(); err != nil {
			o.err = err
			continue
		}

		if _, ok := dbOldM[db.Key()]; ok {
			delete(dbOldM, db.Key())
		}

		dbNew = append(dbNew, db)

		if !ok {
			ok = true
		}
	}
	if ok {
		rs.dbs = dbNew
		for _, v := range dbOldM {
			cleanUp(v)
		}
	}
	return
}

func cleanUp(os ...*Orm) (err error) {
	for _, o := range os {
		t := time.NewTimer(
			time.Duration(int(o.ConnMaxLifetime.Seconds())+1) * time.Second,
		)
		go func() {
			<-t.C
			o.Close()
		}()
	}

	return
}

// ========== /OrmsOpt =========

func News(opts ...OrmsOpt) (dbs *Orms) {
	dbs = &Orms{
		benchmarker: &DefaultBenchmarker{},
		pooler:      &RandomPolicy{},
	}
	dbs.With(opts...)
	return
}

func (m *Orms) With(opts ...OrmsOpt) {
	for _, opt := range opts {
		opt(m)
	}
	if m.write == nil && m.read == nil {
		m.err = errors.New("Please input a instance at least")
	}
}

func (m *Orms) get(c context.Context, dbs *DBs) (db *gorm.DB) {
	dbs.lock.RLock()
	defer dbs.lock.RUnlock()
	return m.pooler.Choose(c, dbs).WithContext(c)
}

func (m *Orms) Read(c context.Context) (db *gorm.DB) {
	if tx, ok := c.Value(contextTransactionKey{}).(*gorm.DB); ok {
		return tx
	}
	if m.benchmarker.Judge(c) {
		if m.shadowRead == nil {
			return m.get(c, m.shadowWrite)
		}
		return m.get(c, m.shadowRead)
	}
	if m.read == nil {
		return m.get(c, m.write)
	}
	return m.get(c, m.read)
}

func (m *Orms) Write(c context.Context) (db *gorm.DB) {
	if tx, ok := c.Value(contextTransactionKey{}).(*gorm.DB); ok {
		return tx
	}
	if m.benchmarker.Judge(c) {
		if m.shadowWrite == nil {
			return m.get(c, m.shadowRead)
		}
		return m.get(c, m.shadowWrite)
	}
	if m.write == nil {
		return m.get(c, m.read)
	}
	return m.get(c, m.write)
}

func (m *Orms) Transaction(c context.Context, fn func(c context.Context) (err error)) error {
	return m.Write(c).Transaction(func(tx *gorm.DB) error {
		if _, ok := c.Value(contextTransactionKey{}).(*gorm.DB); !ok {
			c = context.WithValue(c, contextTransactionKey{}, tx)
		}
		return fn(c)
	})
}

func (m *Orms) Close() func() {
	return func() {
		for _, o := range m.read.dbs {
			o.Close()
		}
		for _, o := range m.write.dbs {
			o.Close()
		}
		for _, o := range m.shadowRead.dbs {
			o.Close()
		}
		for _, o := range m.shadowWrite.dbs {
			o.Close()
		}
	}
}

func (m *Orms) Error() error {
	return m.err
}
