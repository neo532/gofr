package orm

import "time"

/*
 * @abstract Orm's Logger
 * @mail neo532@126.com
 * @date 2024-05-18
 */

type DsnConfig struct {
	Name string `yaml:"name"`
	Dsn  string `yaml:"dsn"`
}

type OrmConfig struct {
	MaxOpenConns        int32         `yaml:"max_open_conns"`
	MaxIdleConns        int32         `yaml:"max_idle_conns"`
	ConnMaxLifetime     time.Duration `yaml:"conn_max_lifetime"`
	MaxSlowtime         time.Duration `yaml:"max_slowtime"`
	TablePrefix         string        `yaml:"table_prefix"`
	Read                []*DsnConfig  `yaml:"read"`
	Write               []*DsnConfig  `yaml:"write"`
	ShadowRead          []*DsnConfig  `yaml:"shadow_read"`
	ShadowWrite         []*DsnConfig  `yaml:"shadow_write"`
	RecordNotFoundError bool          `yaml:"record_not_found_error"`
}
