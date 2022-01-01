package main

import (
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
)

func init() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config/")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("read config failed: %v", err)
	}
}

// User 用户
type User struct {
	Name string
}

func main() {
	dsn := viper.GetString("DbConfig.DSN")
	// 方式一 简单
	// db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	// 方式二 可有更多的自定义配置(数据库驱动程序提供了 一些高级配置 可以在初始化过程中使用)
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn}), &gorm.Config{ // https://gorm.io/zh_CN/docs/gorm_config.html
		SkipDefaultTransaction: false, //跳过默认事务
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",  // 表名前缀
			SingularTable: false, // 使用单数表名
		},
	})

	// Migrator 接口，该接口为每个数据库提供了统一的 API 接口，可用来为您的数据库构建独立迁移
	m := db.Migrator()

	// 反复横跳
	if m.HasTable(&User{}) { // 等价于 m.HasTable("t_users")
		// 删除表
		err = m.DropTable(&User{})
	} else {
		// 建表
		err = m.CreateTable(&User{})
	}

	if err != nil {
		fmt.Println(err.Error())
		return
	}
}
