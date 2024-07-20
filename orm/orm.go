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
func WithRecordNotFoundError(b bool) Opt {
	return func(o *Orm) {
		o.gormLogger.recordNotFoundError = b
	}
}
func WithSlowLog(t time.Duration) Opt {
	return func(o *Orm) {
		o.gormLogger.slowLogTime = t
	}
}
func WithTablePrefix(s string) Opt {
	return func(o *Orm) {
		o.gormOpt.schema.TablePrefix = s
	}
}
func WithLogger(l Logger) Opt {
	return func(o *Orm) {
		o.gormLogger.logger = l
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

	gormLogger *gormLogger
	gormOpt    *gormOpt
	opts       []func(db *sql.DB)
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
		gormLogger: &gormLogger{name: name, logger: NewDefaultLogger()},
		opts:       make([]func(db *sql.DB), 0),
	}
	for _, o := range opts {
		o(db)
	}

	db.orm, db.err = gorm.Open(
		dsn,
		&gorm.Config{
			Logger:         db.gormLogger,
			NamingStrategy: db.gormOpt.schema,
			ClauseBuilders: map[string]clause.ClauseBuilder{
				//hints.Comment("select", "master"),
			},
			//ClauseBuilders: map[string]hints.Comment("select", "master")clause.ClauseBuilder{},
		},
	)

	if db.err != nil {
		db.LogError(name, "Gorm open client error")
		return
	}

	var sqlDB *sql.DB
	if sqlDB, db.err = db.orm.DB(); db.err != nil {
		db.LogError(name, "Orm DB has error")
		return
	}
	for _, o := range db.opts {
		o(sqlDB)
	}

	db.cleanup = func() {
		if sqlDB == nil {
			db.LogError(name, "Close db is nil!")
			return
		}
		if db.err = sqlDB.Close(); db.err != nil {
			db.LogError(name, "Close db has error!")
			return
		}
	}
	ormMap[name] = db
	return
}

func (o *Orm) LogError(name string, message string) {
	o.gormLogger.logger.Error(o.bootstrapContext, message, "name", name, "err", o.err)
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

type gormLogger struct {
	gorm.Config

	name                string
	slowLogTime         time.Duration
	logger              Logger
	recordNotFoundError bool

	LogLevel gLogger.LogLevel
}

func (g *gormLogger) LogMode(level gLogger.LogLevel) gLogger.Interface {
	g.LogLevel = level
	return g
}

func (g *gormLogger) Info(c context.Context, s string, i ...interface{}) {
	g.logger.Info(c, s, i...)
}

func (g *gormLogger) Warn(c context.Context, s string, i ...interface{}) {
	g.logger.Warn(c, s, i...)
}

func (g *gormLogger) Error(c context.Context, s string, i ...interface{}) {
	g.logger.Error(c, s, i...)
}

func (g *gormLogger) Trace(c context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	cost := time.Since(begin)

	if err == gorm.ErrRecordNotFound && !g.recordNotFoundError {
		err = nil
	}

	p := []interface{}{
		"name", g.name,
		"limit", g.slowLogTime,
		"cost", cost,
		"rows", rows,
	}

	if err != nil {
		p = append(p, "err", err)
		g.logger.Error(c, sql, p...)
		return
	}

	if cost > g.slowLogTime {
		g.logger.Warn(c, "[slow]"+sql, p...)
		return
	}

	g.logger.Info(c, sql, p...)
}
