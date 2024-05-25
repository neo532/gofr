package orm

/*
 * @abstract Orm client
 * @mail neo532@126.com
 * @date 2024-05-18
 */

import (
	"context"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/mysql"
)

type DB struct {
	Name            string
	Dsn             string
	TablePrefix     string
	ConnMaxLifetime time.Duration
	MaxIdleConns    int
	MaxOpenConns    int
	MaxSlowtime     time.Duration
}

type DefaultDBConf struct {
	Read        *DB
	Write       *DB
	ShadowRead  *DB
	ShadowWrite *DB
}

func connect(c context.Context, cfg *DB, logger Logger) *Orm {
	return New(
		cfg.Name,
		mysql.Open(cfg.Dsn),

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

var (
	dbs   *Orms
	clean func()
)

func init() {
	dsn := "user:passwd@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=true&loc=Local"
	d := &DefaultDBConf{
		Read:        &DB{Name: "default_read", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
		Write:       &DB{Name: "default_write", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
		ShadowRead:  &DB{Name: "default_shadowread", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
		ShadowWrite: &DB{Name: "default_shadowwrite", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
	}
	logger := NewDefaultLogger()
	c := context.Background()

	opts := make([]OrmsOpt, 0, 4)
	if d.Read != nil {
		opts = append(opts, WithRead(connect(c, d.Read, logger)))
	}
	if d.Write != nil {
		opts = append(opts, WithWrite(connect(c, d.Write, logger)))
	}
	if d.ShadowRead != nil {
		opts = append(opts, WithShadowRead(connect(c, d.ShadowRead, logger)))
	}
	if d.ShadowWrite != nil {
		opts = append(opts, WithShadowWrite(connect(c, d.ShadowWrite, logger)))
	}
	dbs = News(opts...)
	clean = dbs.Cleanup()
}

func TestOrms(t *testing.T) {

	//dbs, _ := orms()
	//defer clean()

	c := context.Background()
	var databases []string
	var err error
	if err = dbs.Write(c).Raw("show databases").Scan(&databases).Error; err != nil {
		t.Errorf("%s has err[%+v]", t.Name(), err)
	}
	fmt.Println(fmt.Sprintf("dbs:%+v\t%+v", databases, err))

	if err = dbs.Read(c).Raw("show databases").Scan(&databases).Error; err != nil {
		t.Errorf("%s has err[%+v]", t.Name(), err)
	}
	fmt.Println(fmt.Sprintf("dbs:%+v\t%+v", databases, err))
}

func TestTransaction(t *testing.T) {

	//dbs, _ := orms()
	//defer clean()

	c := context.Background()
	err := dbs.Transaction(c, func(ctx context.Context) (err error) {

		var databases []string

		// Please notice this ctx not c.
		if err = dbs.Write(ctx).Raw("show databases").Scan(&databases).Error; err != nil {
			return
		}

		// Please notice this ctx not c.
		if err = dbs.Read(ctx).Raw("show databases").Scan(&databases).Error; err != nil {
			return
		}
		fmt.Println(fmt.Sprintf("txdbs:%+v", databases))
		return
	})
	if err != nil {
		t.Errorf("%s has err[%+v]", t.Name(), err)
	}
}
