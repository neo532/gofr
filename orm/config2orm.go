package orm

import (
	"context"

	"gorm.io/driver/mysql"
)

/*
 * @abstract Orm's Logger
 * @mail neo532@126.com
 * @date 2024-05-18
 */

func Connect(c context.Context, cfg *OrmConfig, dsn *DsnConfig, logger Logger) *Orm {
	return New(
		dsn.Name,
		mysql.Open(dsn.Dsn),
		WithTablePrefix(cfg.TablePrefix),
		WithConnMaxLifetime(cfg.ConnMaxLifetime),
		WithMaxIdleConns(cfg.MaxIdleConns),
		WithMaxOpenConns(cfg.MaxOpenConns),
		WithLogger(logger),
		WithSingularTable(),
		WithContext(c),
		WithSlowLog(cfg.MaxSlowtime),
	)
}

func NewOrms(c context.Context, d *OrmConfig, logger Logger) (dbs *Orms, clean func(), err error) {
	opts := make([]OrmsOpt, 0, 4)
	if d.Read != nil {
		for _, dsn := range d.Read {
			opts = append(opts, WithRead(Connect(c, d, dsn, logger)))
		}
	}
	if d.Write != nil {
		for _, dsn := range d.Write {
			opts = append(opts, WithWrite(Connect(c, d, dsn, logger)))
		}
	}
	if d.ShadowRead != nil {
		for _, dsn := range d.ShadowRead {
			opts = append(opts, WithShadowRead(Connect(c, d, dsn, logger)))
		}
	}
	if d.ShadowWrite != nil {
		for _, dsn := range d.ShadowWrite {
			opts = append(opts, WithShadowWrite(Connect(c, d, dsn, logger)))
		}
	}
	dbs = News(opts...)
	clean = dbs.Cleanup()
	return
}
