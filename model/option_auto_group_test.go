package model

import (
	"maps"
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptionValueRejectsInvalidMaxTokenAutoGroups(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "1.5", "invalid"} {
		t.Run(value, func(t *testing.T) {
			assert.Error(t, validateOptionValue("MaxTokenAutoGroups", value))
		})
	}
	require.NoError(t, validateOptionValue("MaxTokenAutoGroups", "999999"))
}

func TestHiddenGroupRatioOptionsDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector = sqlite.Open(":memory:")
			if dialect != "sqlite" {
				env := "TEST_MYSQL_DSN"
				if dialect == "postgres" {
					env = "TEST_POSTGRES_DSN"
				}
				dsn := os.Getenv(env)
				if dsn == "" {
					t.Skip(env + " not configured")
				}
				if dialect == "mysql" {
					driver = mysql.Open(dsn)
				} else {
					driver = postgres.Open(dsn)
				}
			}
			db, err := gorm.Open(driver, &gorm.Config{NamingStrategy: schema.NamingStrategy{TablePrefix: "hidden_ratio_test_"}})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			savedConfig := map[string]string{}
			require.NoError(t, config.GlobalConfig.SaveToDB(func(k, v string) error { savedConfig[k] = v; return nil }))
			oldDB, oldType := DB, common.MainDatabaseType()
			common.OptionMapRWMutex.Lock()
			oldOptions := common.OptionMap
			common.OptionMap = maps.Clone(common.OptionMap)
			if common.OptionMap == nil {
				common.OptionMap = make(map[string]string)
			}
			common.OptionMapRWMutex.Unlock()
			DB = db
			common.SetMainDatabaseType(common.DatabaseType(dialect))
			t.Cleanup(func() {
				require.NoError(t, db.Migrator().DropTable(&Task{}, &Option{}))
				DB = oldDB
				common.SetMainDatabaseType(oldType)
				require.NoError(t, config.GlobalConfig.LoadFromDB(savedConfig))
				common.OptionMapRWMutex.Lock()
				common.OptionMap = oldOptions
				common.OptionMapRWMutex.Unlock()
				require.NoError(t, sqlDB.Close())
			})
			versionSQL := "SELECT version()"
			if dialect == "sqlite" {
				versionSQL = "SELECT sqlite_version()"
			}
			var version string
			require.NoError(t, db.Raw(versionSQL).Scan(&version).Error)
			t.Logf("database version: %s", version)
			require.NoError(t, db.AutoMigrate(&Option{}, &Task{}))
			legacy := Task{TaskID: "legacy", PrivateData: TaskPrivateData{BillingContext: &TaskBillingContext{GroupRatio: 1}}}
			require.NoError(t, db.Create(&legacy).Error)
			require.NoError(t, UpdateOption("GroupRatio", `{"default":0.8}`))
			key := ratio_setting.HiddenGroupRatioOptionKey
			require.NoError(t, UpdateOption(key, `{"default":1.25}`))
			info := ratio_setting.ResolveBillingGroupRatio("default", "default")
			assert.Equal(t, 1.25, info.GroupRatio)
			assert.Equal(t, 0.8, info.PublicRatio())
			for _, invalid := range []string{"null", "[]", "1", `{"default":null}`, `{"default":-1}`, `{"default":"2"}`, `{"default":1e999}`, `{"":1}`} {
				assert.Error(t, UpdateOption(key, invalid), invalid)
				assert.Equal(t, 1.25, ratio_setting.ResolveBillingGroupRatio("default", "default").GroupRatio)
			}
			require.NoError(t, UpdateOption(key, `{"default":0}`))
			info = ratio_setting.ResolveBillingGroupRatio("default", "default")
			task := Task{TaskID: "hidden-free", PrivateData: TaskPrivateData{BillingContext: &TaskBillingContext{GroupRatio: info.GroupRatio, GroupRatioInfo: &info}}}
			require.NoError(t, db.Create(&task).Error)
			for range 2 {
				require.NoError(t, db.AutoMigrate(&Option{}, &Task{}))
				var stored Option
				require.NoError(t, db.Where(Option{Key: key}).First(&stored).Error)
				require.JSONEq(t, `{"default":0}`, stored.Value)
				require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{key: stored.Value}))
				assert.Equal(t, 0.0, ratio_setting.ResolveBillingGroupRatio("default", "default").GroupRatio)
				var restored Task
				require.NoError(t, db.First(&restored, task.ID).Error)
				require.NotNil(t, restored.PrivateData.BillingContext)
				snapshot := restored.PrivateData.BillingContext.GroupRatioInfo
				require.NotNil(t, snapshot)
				assert.Equal(t, 0.0, snapshot.GroupRatio)
				assert.Equal(t, 0.8, snapshot.PublicRatio())
				publicJSON, err := common.Marshal(restored)
				require.NoError(t, err)
				assert.NotContains(t, string(publicJSON), "group_ratio_info")
				assert.NotContains(t, string(publicJSON), "DisplayGroupRatio")
				restored = Task{}
				require.NoError(t, db.First(&restored, legacy.ID).Error)
				assert.Nil(t, restored.PrivateData.BillingContext.GroupRatioInfo)
			}
			require.NoError(t, UpdateOption(key, `{}`))
			assert.Equal(t, 0.8, ratio_setting.ResolveBillingGroupRatio("default", "default").GroupRatio)
		})
	}
}
