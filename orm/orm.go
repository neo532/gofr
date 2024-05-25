package orm

/*
 * @abstract Orm client
 * @mail neo532@126.com
 * @date 2024-05-18
 */

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	//"gorm.io/gorm/hints"
	gLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var (
	instanceLock sync.Mutex
	ormMap       = make(map[string]*Orm, 2)
)

// ========== Option ==========
type gormOpt struct {
	schema schema.NamingStrategy
}

type Opt func(*Orm)

func WithMaxIdleConns(i int) Opt {
	return func(o *Orm) {
		o.opts = append(o.opts, func(db *sql.DB) {
			db.SetMaxIdleConns(i)
		})
	}
}
func WithMaxOpenConns(i int) Opt {
	return func(o *Orm) {
		o.opts = append(o.opts, func(db *sql.DB) {
			db.SetMaxOpenConns(i)
		})
	}
}
func WithConnMaxLifetime(t time.Duration) Opt {
	return func(o *Orm) {
		o.opts = append(o.opts, func(db *sql.DB) {
			db.SetConnMaxLifetime(t)
		})
	}
}
func WithSlowLog(t time.Duration) Opt {
	return func(o *Orm) {
		o.slowTime = t
	}
}
func WithTablePrefix(s string) Opt {
	return func(o *Orm) {
		o.gormOpt.schema.TablePrefix = s
	}
}
func WithLogger(l Logger) Opt {
	return func(o *Orm) {
		o.logger = l
	}
}
func WithSingularTable() Opt {
	return func(o *Orm) {
		o.gormOpt.schema.SingularTable = true
	}
}
func WithContext(c context.Context) Opt {
	return func(o *Orm) {
		o.bootstrapContext = c
	}
}

// ========== /Option ==========
type Orm struct {
	orm              *gorm.DB
	cleanup          func()
	err              error
	bootstrapContext context.Context

	gormOpt  *gormOpt
	opts     []func(db *sql.DB)
	slowTime time.Duration
	logger   Logger
}

// New returns a instance of Orm.
// this Name must be unique to special instance.
func New(name string, dsn gorm.Dialector, opts ...Opt) (db *Orm) {
	instanceLock.Lock()
	defer instanceLock.Unlock()

	var ok bool
	if db, ok = ormMap[name]; ok {
		return
	}

	db = &Orm{
		bootstrapContext: context.Background(),
		gormOpt: &gormOpt{
			schema: schema.NamingStrategy{},
		},
		opts: make([]func(db *sql.DB), 0),
	}
	for _, o := range opts {
		o(db)
	}

	db.orm, db.err = gorm.Open(
		dsn,
		&gorm.Config{
			Logger:         NewGormLogger(name, db.slowTime, db.logger),
			NamingStrategy: db.gormOpt.schema,
			ClauseBuilders: map[string]clause.ClauseBuilder{
				//hints.Comment("select", "master"),
			},
			//ClauseBuilders: map[string]hints.Comment("select", "master")clause.ClauseBuilder{},
		},
	)

	if db.err != nil {
		db.logger.
			Errorf(db.bootstrapContext, "Gorm open client[%s] error: %+v",
				name,
				db.err,
			)
		return
	}

	var sqlDB *sql.DB
	if sqlDB, db.err = db.orm.DB(); db.err != nil {
		db.logger.
			Errorf(db.bootstrapContext, "Orm DB[%s] has error: %+v",
				name,
				db.err,
			)
		return
	}
	for _, o := range db.opts {
		o(sqlDB)
	}

	db.cleanup = func() {
		if sqlDB == nil {
			db.logger.
				Errorf(db.bootstrapContext, "Close db[%s] is nil!", name)
			return
		}
		if db.err = sqlDB.Close(); db.err != nil {
			db.logger.
				Errorf(db.bootstrapContext, "Close db[%s] has error: %+v", name, db.err)
			return
		}
	}
	ormMap[name] = db
	return
}

func (o *Orm) Error() error {
	return o.err
}

func (o *Orm) Orm() *gorm.DB {
	return o.orm
}

func (o *Orm) Cleanup() func() {
	return o.cleanup
}

type GormLogger struct {
	gorm.Config

	db          string
	slowLogTime time.Duration
	logger      Logger

	LogLevel gLogger.LogLevel
}

func NewGormLogger(db string, slowLogTime time.Duration, logger Logger) *GormLogger {
	return &GormLogger{
		db:          db,
		slowLogTime: slowLogTime,
		logger:      logger,
	}
}

func (g *GormLogger) LogMode(level gLogger.LogLevel) gLogger.Interface {
	g.LogLevel = level
	return g
}

func (g *GormLogger) Info(c context.Context, s string, i ...interface{}) {
	g.logger.Infof(c, s, i...)
}

func (g *GormLogger) Warn(c context.Context, s string, i ...interface{}) {
	g.logger.Warnf(c, s, i...)
}

func (g *GormLogger) Error(c context.Context, s string, i ...interface{}) {
	g.logger.Errorf(c, s, i...)
}

func (g *GormLogger) Trace(c context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	cost := time.Since(begin)

	if err == gorm.ErrRecordNotFound {
		err = nil
	}

	if err != nil {
		g.logger.
			Errorf(c, "err:[%+v], name:%s, limit:%v, cost:%v, rows:%d, sql:[%s]",
				err,
				g.db,
				g.slowLogTime,
				cost,
				rows,
				sql,
			)
		return
	}

	if cost > g.slowLogTime {
		g.logger.
			Warnf(c, "slowlog, name:%s, limit:%v, cost:%v, rows:%d, sql:[%s]",
				g.db,
				g.slowLogTime,
				cost,
				rows,
				sql,
			)
		return
	}

	g.logger.
		Infof(c, "name:%s, limit:%v, cost:%s, rows:%d, sql:[%s]",
			g.db,
			g.slowLogTime,
			cost,
			rows,
			sql,
		)
}
