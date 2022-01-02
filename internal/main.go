package main

import (
	"database/sql"
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"log"
	"time"
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
// GORM 倾向于约定(https://gorm.io/zh_CN/docs/conventions.html)，而不是配置。默认情况下，GORM 使用 ID 作为主键，
// 使用结构体名的 蛇形复数 作为表名，字段名的 蛇形 作为列名，并使用 CreatedAt、UpdatedAt 字段追踪创建、更新时间
type User struct {
	ID           uint
	Name         string
	Email        *string `gorm:"default:default@gmail.com"`
	Age          uint8
	Birthday     *time.Time
	MemberNumber sql.NullString
	ActivatedAt  sql.NullTime
	// GORM 约定使用 CreatedAt、UpdatedAt 追踪创建/更新时间。如果您定义了这种字段，GORM 在创建、更新时会自动填充 当前时间
	// 如果想要保存 UNIX（毫/纳）秒时间戳，而不是 time，只需简单地将 time.Time 修改为 int 即可
	// CreatedAt time.Time
	CreatedAt int64 `gorm:"autoCreateTime"`
	// UpdatedAt time.Time
	// 要使用不同名称的字段，您可以配置 autoCreateTime、autoUpdateTime 标签
	UpdateOn int64 `gorm:"autoUpdateTime"`
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
	/*	if !m.HasTable(&User{}) { // 等价于 m.HasTable("t_users")
			// 删除表
			err = m.DropTable(&User{})
		} else {
			// 建表
			err = m.CreateTable(&User{})
		}*/
	err = initTable(m)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Test CRUD
	testCreate(db)
}

func initTable(m gorm.Migrator) error {
	if !m.HasTable(&User{}) {
		err := m.CreateTable(&User{})
		if err != nil {
			return err
		}
	}
	return nil
}

func testCreate(gormDb *gorm.DB) {
	// clear table
	now := time.Now()
	user := User{
		Name:     "sharpe-x",
		Age:      18,
		Birthday: &now,
	}

	// insert record
	result := gormDb.Create(&user)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	mail := "test@gmail.com"
	user2 := User{
		Name:     "sharpe-x-2",
		Age:      19,
		Birthday: &now,
		Email:    &mail,
	}

	//创建记录并更新给出的字段
	// Birthday 会被忽略
	// INSERT INTO `t_users` (`name`,`age`,`create_on`) VALUES ("sharpe-x-2", 19, 1641103780)
	result = gormDb.Select("Name", "Age", "UpdateOn").Create(&user2)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	user3 := User{
		Name:     "sharpe-x-3",
		Age:      19,
		Birthday: &now,
	}
	// 创建一个记录且一同忽略传递给略去的字段值。
	// Name Age UpdateOn 被忽略
	// INSERT INTO `t_users` (`email`,`birthday`,`member_number`,`activated_at`,`created_at`,`update_on`,`id`) VALUES (NULL,'2022-01-02 14:12:17.739',NULL,NULL,1641103937,1641103937,7)
	result = gormDb.Omit("Name", "Age", "UpdateOn").Create(&user3)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	// batch insert
	var users = []User{
		{
			Name: "sharpe1",
		},
		{
			Name: "sharpe2",
		},
		{
			Name: "sharpe3",
		},
	}
	result = gormDb.Create(&users)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	// batch insert
	var batchUsers = []User{
		{
			Name: "sharpe-batches-1",
		},
		{
			Name: "sharpe-batches-2",
		},
		{
			Name: "sharpe-batches-3",
		},
		{
			Name: "sharpe-batches-4",
		},
		{
			Name: "sharpe-batches-5",
		},
		{
			Name: "sharpe-batches-6",
		},
	}

	// 分批创建
	result = gormDb.CreateInBatches(batchUsers, 2)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	skipUser := &User{
		Name:  "sharpe-skip-hook",
		Email: &mail,
	}

	// 跳过钩子方法 skipUser age 是0 不是20
	result = gormDb.Session(&gorm.Session{SkipHooks: true}).Create(&skipUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	// 根据Map 创建 使用map 创建时 时间不会自动更新 钩子函数也不会触发 association 不会被调用，且主键也不会自动填充
	// GORM 支持根据 map[string]interface{} 和 []map[string]interface{}{} 创建记录
	result = gormDb.Model(&User{}).Create(map[string]interface{}{
		"Name": "sharpe-map", "Age": 28,
	})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	// 根据Map 批量创建
	result = gormDb.Model(&User{}).CreateInBatches([]map[string]interface{}{
		{"Name": "sharpe-map-batches-1", "Age": 23},
		{"Name": "sharpe-map-batches-2", "Age": 24},
		{"Name": "sharpe-map-batches-3", "Age": 25, "UpdateOn": time.Now().Unix()},
		{"Name": "sharpe-map-batches-4", "CreatedAt": time.Now().Unix()},
	}, 2)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	// 使用 SQL 表达式、Context Valuer 创建记录
	// 关联创建
	// TODO

	// 默认值
	//标签 default 为字段定义默认值
	// `gorm:"default:default@gmail.com"`
	// 插入记录到数据库时，默认值 会被用于 填充值为 零值 的字段

	// Upsert 及冲突
	// TODO
}

// BeforeCreate https://gorm.io/zh_CN/docs/hooks.html hook 函数
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.Age == 0 {
		u.Age = 20
	}
	return
}
