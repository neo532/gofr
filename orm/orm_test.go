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

func getConfig() (d *DefaultDBConf) {
	dsn := "root:12345678@tcp(127.0.0.1:3306)/test?charset=utf8mb4&parseTime=true&loc=Local"
	d = &DefaultDBConf{
		Read:        &DB{Name: "default_read", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
		Write:       &DB{Name: "default_write", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
		ShadowRead:  &DB{Name: "default_shadowread", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
		ShadowWrite: &DB{Name: "default_shadowwrite", Dsn: dsn, ConnMaxLifetime: 3 * time.Second, MaxIdleConns: 2, MaxOpenConns: 2, MaxSlowtime: 3 * time.Second},
	}
	return
}

func init() {
	logger := NewDefaultLogger()
	c := context.Background()
	var opts []OrmsOpt
	var d *DefaultDBConf

	d = getConfig()
	opts = []OrmsOpt{
		WithRead(connect(c, d.Read, logger)),
		WithWrite(connect(c, d.Write, logger)),
		WithShadowRead(connect(c, d.ShadowRead, logger)),
		WithShadowWrite(connect(c, d.ShadowWrite, logger)),
	}
	dbs = News(opts...)

	d = getConfig()
	opts = []OrmsOpt{
		WithRead(connect(c, d.Read, logger)),
		WithWrite(connect(c, d.Write, logger)),
		WithShadowRead(connect(c, d.ShadowRead, logger)),
		WithShadowWrite(connect(c, d.ShadowWrite, logger)),
	}
	dbs.With(opts...)
	clean = dbs.Close()
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
	time.Sleep(10 * time.Second)
}

func TestTransaction(t *testing.T) {

	//dbs, _ := orms()
	//defer clean()

	c := context.Background()
	err := dbs.Transaction(c, func(c context.Context) (err error) {

		var databases []string

		if err = dbs.Write(c).Raw("show databases").Scan(&databases).Error; err != nil {
			return
		}

		if err = dbs.Read(c).Raw("show databases").Scan(&databases).Error; err != nil {
			return
		}
		fmt.Println(fmt.Sprintf("txdbs:%+v", databases))
		return
	})
	if err != nil {
		t.Errorf("%s has err[%+v]", t.Name(), err)
	}
}
