package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/soft_delete"
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
	UpdateOn  int64                 `gorm:"autoUpdateTime"`
	IsDeleted soft_delete.DeletedAt `gorm:"softDelete:flag default:0"`
}

func initTable(m gorm.Migrator) error {
	if !m.HasTable(&User{}) {
		err := m.CreateTable(&User{})
		if err != nil {
			return err
		}
	}

	return m.AutoMigrate(&User{})
}

// BeforeCreate https://gorm.io/zh_CN/docs/hooks.html hook 函数
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.Age == 0 {
		u.Age = 20
	}
	return
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
	//testCreate(db)
	//testQuery(db)
	//testUpdate(db)
	//testDelete(db)
	testTransaction(db)
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

func testQuery(gormDb *gorm.DB) {

	firstUser := &User{}
	// 获取第一条记录（主键升序）
	result := gormDb.First(&firstUser)
	if result.Error != nil {

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("First RecordNotFound")
			return
		}

		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("firstUser = %+v\n", firstUser)

	takeUser := new(User)
	// 获取一条记录，没有指定排序字段
	result = gormDb.Take(&takeUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("Take RecordNotFound")
			return
		}
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("takeUser = %+v\n", takeUser)

	lastUser := new(User)
	// 获取最后一条记录（主键降序)
	result = gormDb.Last(&lastUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("Last RecordNotFound")
			return
		}
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)
	// First 和 Last 会根据主键排序，分别查询第一条和最后一条记录。 只有在目标 struct 是指针或者通过 db.Model() 指定 model 时，该方法才有效。
	//	此外，如果相关 model 没有定义主键，那么将按 model 的第一个字段进行排序
	// 如果你想避免ErrRecordNotFound错误，你可以使用Find，比如db.Limit(1).Find(&user)，Find方法可以接受struct和slice的数据。

	userMap := map[string]interface{}{}
	// 通过 db.Model() 指定 model
	result = gormDb.Model(&User{}).Last(&userMap)
	// gormDb.Table("users").First(&result) 这种用法在 First 和 Last是 不可以的!!!!!! 但是take 可以
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("Model Last RecordNotFound")
			return
		}
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("userMap = %+v\n", userMap)

	// 用主键检索
	primaryKeyUser := new(User)
	result = gormDb.First(&primaryKeyUser, 10)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("primaryKey = 10 is RecordNotFound")
		} else {
			fmt.Println(result.Error.Error())
			return
		}
	}

	result = gormDb.First(&primaryKeyUser, 204)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("Model Last RecordNotFound")
			return
		}
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("primaryKeyUser = %+v\n", primaryKeyUser)

	var users []User
	result = gormDb.Find(&users, []int{100, 101, 102})
	if result.Error != nil {
		// Find 会避免ErrRecordNotFound错误
		/*		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				fmt.Println("Find users RecordNotFound")
				return
			}*/
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("users = %+v\n", users)

	result = gormDb.Find(&users, []int{200, 201, 202})
	// SELECT * FROM users WHERE id IN (200,201,202);
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("users[%d] = %+v\n", len(users), users)

	firstUserById := new(User)
	result = gormDb.First(&firstUserById, "id = ?", 206)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("Find users RecordNotFound")
			return
		}
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("firstUserById = %+v\n", firstUserById)

	var allUser []User
	//获取全部记录
	result = gormDb.Find(&allUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("allUser[%d] = %+v\n", len(allUser), allUser)

	// String 条件
	var whereFirstUser User
	// 获取第一条匹配的记录
	result = gormDb.Where("name = ?", "sharpe-batches-3").First(&whereFirstUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("Find users RecordNotFound")
			return
		}
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("whereFirstUser= %+v\n", whereFirstUser)

	var whereAllUser []*User
	// 获取全部匹配的记录
	result = gormDb.Where("name <> ?", "sharpe-batches-3").Find(&whereAllUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("whereAllUser len =  %d\n", len(whereAllUser))

	// 获取全部匹配的记录
	result = gormDb.Where("name in ?", []string{"sharpe-batches-1", "sharpe-skip-hook"}).Find(&whereAllUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("whereAllUser len =  %d\n", len(whereAllUser))

	/*// LIKE
	db.Where("name LIKE ?", "%jin%").Find(&users)
	// SELECT * FROM users WHERE name LIKE '%jin%';

	// AND
	db.Where("name = ? AND age >= ?", "jinzhu", "22").Find(&users)
	// SELECT * FROM users WHERE name = 'jinzhu' AND age >= 22;

	// Time
	db.Where("updated_at > ?", lastWeek).Find(&users)
	// SELECT * FROM users WHERE updated_at > '2000-01-01 00:00:00';

	// BETWEEN
	db.Where("created_at BETWEEN ? AND ?", lastWeek, today).Find(&users)*/

	// Struct & Map 条件
	result = gormDb.Where(&User{Age: 20}).Find(&whereAllUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("whereAllUser len =  %d\n", len(whereAllUser))

	// 实际上查到了全部 主要是当使用结构作为条件查询时，GORM 只会查询非零值字段。这意味着如果您的字段值为 0、''、false 或其他 零值
	// 如果想要包含零值查询条件，你可以使用 map，其会包含所有 key-value 的查询条件
	result = gormDb.Where(&User{Age: 0}).Find(&whereAllUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("whereAllUser len =  %d\n", len(whereAllUser))

	filters := map[string]interface{}{
		"Age":  0,
		"Name": "sharpe-skip-hook",
	}

	// 得到预期
	result = gormDb.Where(filters).Find(&whereAllUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("whereAllUser len =  %d\n", len(whereAllUser))

	// 指定结构体查询字段
	// 等价于 SELECT * FROM t_users WHERE age = 0;
	var ageUser User
	result = gormDb.Where(&User{
		Name: "lala",
	}, "Age").First(&ageUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("whereFirstUser =  %v\n", ageUser)
	// 查询条件也可以被内联到 First 和 Find 之类的方法中，其用法类似于 Where。

	var findAllUser []User
	result = gormDb.Find(&findAllUser, filters)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("findAllUser len =  %d , findAllUser = %v\n", len(findAllUser), findAllUser)
	// Not 条件 用法与 Where 类似
	// Or 条件
	/*db.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&users)
	// SELECT * FROM users WHERE role = 'admin' OR role = 'super_admin';

	// Struct
	db.Where("name = 'jinzhu'").Or(User{Name: "jinzhu 2", Age: 18}).Find(&users)
	// SELECT * FROM users WHERE name = 'jinzhu' OR (name = 'jinzhu 2' AND age = 18);

	// Map
	db.Where("name = 'jinzhu'").Or(map[string]interface{}{"name": "jinzhu 2", "age": 18}).Find(&users)
	// SELECT * FROM users WHERE name = 'jinzhu' OR (name = 'jinzhu 2' AND age = 18);*/

	var selectUser []User
	// Select 允许从数据库中检索哪些字段， 默认情况下，GORM 会检索所有字段。
	result = gormDb.Select("name").Where(filters).Find(&selectUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	for i, user := range selectUser {
		fmt.Printf("%d := %+v\n", i, user)
	}

	/*db.Select([]string{"name", "age"}).Find(&users)
	// SELECT name, age FROM users;

	db.Table("users").Select("COALESCE(age,?)", 42).Rows()
	// SELECT COALESCE(age,'42') FROM users;*/

	// Order
	orderUser := make([]User, 0)
	result = gormDb.Order("age desc, name").Find(&orderUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("orderUser len =  %d , orderUser[0] = %v\n", len(orderUser), orderUser[0])

	// Order
	orderUser2 := make([]User, 0)
	result = gormDb.Order("age desc").Order("name").Find(&orderUser2)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("orderUser2 len =  %d , orderUser2[0] = %v\n", len(orderUser2), orderUser2[0])

	// Limit & Offset
	orderUser3 := make([]User, 0)
	result = gormDb.Limit(10).Offset(5).Find(&orderUser3)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("orderUser3 len =  %d , orderUser3[0] = %v\n", len(orderUser3), orderUser3[0])

	// Group By & Having &Distinct &Joins
	// Todo

	// Scan

	var names []string

	result = gormDb.Table("t_users").Select("name").Where("name != ?", "").Find(&names)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("len(names) = %d,names = %v\n", len(names), names)

	var names2 []string
	result = gormDb.Table("t_users").Select("name").Find(&names2, "name != ?", "")
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("len(names2) = %d,names2 = %v\n", len(names2), names2)

	var names3 []string
	result = gormDb.Raw("SELECT name FROM t_users WHERE name != ?", "").Find(&names3)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("len(names3) = %d,names3 = %v\n", len(names3), names3)

	var names4 []string
	result = gormDb.Raw("SELECT name FROM t_users WHERE name != ?", "haha").Find(&names3)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("len(names4) = %d,names4 = %v\n", len(names4), names4)

	var ages []string
	result = gormDb.Raw("SELECT age FROM t_users WHERE age is not null").Find(&ages)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("len(ages) = %d,ages = %v\n", len(ages), ages)

	var ages1 []string
	result = gormDb.Table("t_users").Select("age").Distinct("age").Find(&ages1, "age is not null")
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("len(ages1) = %d,ages1 = %v\n", len(ages1), ages1)

	// 高级查询
	// TODO

}

func testUpdate(gormDb *gorm.DB) {

	firstUser := &User{}
	// 获取第一条记录（主键升序）
	result := gormDb.First(&firstUser)
	if result.Error != nil {

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("First RecordNotFound")
			return
		}

		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("firstUser = %+v\n", firstUser)
	firstUser.Age = 100

	// Save 会保存所有的字段，即使字段是零值
	//  UPDATE `t_users` SET `name`='sharpe-x',`email`='default@gmail.com',`age`=100,`birthday`='2022-01-02 16:53:41.544',`member_number`=NULL,`activated_at`=NULL,`created_at`=1641113621,`update_on`=1641213885 WHERE `id` = 200
	result = gormDb.Save(firstUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	// update
	//  UPDATE `t_users` SET `age`=25,`update_on`=1641214140 WHERE age = 20
	result = gormDb.Model(&User{}).Where("age = ?", 20).Update("age", 25)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	lastUser := new(User)
	result = gormDb.Last(&lastUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	result = gormDb.Model(&lastUser).Update("age", 66)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	//  UPDATE `t_users` SET `birthday`='2022-01-03 20:57:03.746',`update_on`=1641214623 WHERE Birthday is null AND `id` = 217
	// 根据条件和 model 的值进行更新
	result = gormDb.Model(&lastUser).Where("birthday is null").Update("birthday", time.Now())
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	// 更新多列
	// 当使用 struct 更新时，默认情况下，GORM 只会更新非零值的字段
	// UPDATE `t_users` SET `name`='hello-update',`email`='hello-update@gmail.com',`update_on`=1641214897 WHERE `id` = 217
	mail := "hello-update@gmail.com"
	result = gormDb.Model(&lastUser).Updates(User{
		Name:  "hello-update",
		Email: &mail,
	})

	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	// 根据 `map` 更新属性
	//  UPDATE `t_users` SET `age`=0,`birthday`='2021-01-03 21:05:06.072',`name`='hello-update-map',`update_on`=1641215106 WHERE `id` = 217
	result = gormDb.Model(&lastUser).Updates(
		map[string]interface{}{
			"name":     "hello-update-map",
			"age":      0,
			"birthday": time.Now().AddDate(-1, 0, 0),
		})

	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	result = gormDb.Model(&lastUser).Select("name").Updates(map[string]interface{}{"name": "hello-updates-select",
		"age": 18})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	result = gormDb.Model(&lastUser).Omit("name").Updates(map[string]interface{}{"name": "hello-updates-select2",
		"age": 18})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	// Select 除 email 外的所有字段（包括零值字段的所有字段）
	b10year := time.Now().AddDate(-10, 0, 0)
	result = gormDb.Model(&lastUser).Select("*").Omit("email", "id").Updates(User{
		Name:     "1223333",
		Age:      0,
		Birthday: &b10year,
	})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("lastUser = %+v\n", lastUser)

	// GORM 支持 BeforeSave、BeforeUpdate、AfterSave、AfterUpdate hook
	// Todo

	// 批量更新
	result = gormDb.Model(User{}).Where("age is not null").Updates(User{Age: 18})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	result = gormDb.Model(User{}).Where("age is null").Updates(User{Age: 18})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	// 阻止全局更新
	//在没有任何条件的情况下执行批量更新，默认情况下，GORM 不会执行该操作，并返回 ErrMissingWhereClause 错误
	// 对此，你必须加一些条件，或者使用原生 SQL，或者启用 AllowGlobalUpdate 模式，例如
	result = gormDb.Model(&User{}).Where("1 = 1").Updates(User{
		Age: 19,
	})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	result = gormDb.Exec("UPDATE t_users SET age = ?", 23)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	result = gormDb.Session(&gorm.Session{
		AllowGlobalUpdate: true,
	}).Model(&User{}).Updates(User{
		Age:      35,
		Birthday: &b10year,
	})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("RowsAffected = %d\n", result.RowsAffected)

	//使用 SQL 表达式更新
	// Todo
}

func testDelete(gormDb *gorm.DB) {
	firstUser := new(User)
	result := gormDb.Model(&User{}).First(&firstUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			fmt.Println("First RecordNotFound")
		} else {
			fmt.Println(result.Error.Error())
			return
		}
	}

	fmt.Printf("firstUser = %+v \n", firstUser)

	//删除一条记录时，删除对象需要指定主键
	result = gormDb.Where("is_deleted = ?", 0).Delete(firstUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("after delete firstUser = %+v \n", firstUser)

	//DELETE FROM `t_users` WHERE name = 'sharpe-x-2' AND `t_users`.`id` = 3
	/*result = gormDb.Where("name = ?", "sharpe-x-2").Delete(&firstUser)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}*/

	// 根据主键删除
	// UPDATE `t_users` SET `is_deleted`=1641371984 WHERE `t_users`.`id` = 10 AND `t_users`.`is_deleted` = 0
	result = gormDb.Delete(&User{}, 10)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	// 批量删除
	result = gormDb.Delete(&User{}, "is_deleted = ?", 0)
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}

	result = gormDb.Model(&User{}).Unscoped().Where("is_deleted != ?", 0).Updates(map[string]interface{}{
		"is_deleted": 0,
	})
	if result.Error != nil {
		fmt.Println(result.Error.Error())
		return
	}
	fmt.Printf("%d\n", result.RowsAffected)

}

func testTransaction(gormDb *gorm.DB) {
	err := gormDb.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&User{Age: 19, Name: "hello-transaction"}).Error; err != nil {
			return err
		}

		if err := tx.Create(&User{Age: 20, Name: "hello-transaction2"}).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		fmt.Printf("err = %v\n", err)
		return
	}

	// 嵌套事务
	// Todo

	// 手动控制事务
	tx := gormDb.Begin()
	err = transaction(tx)
	if err != nil {
		fmt.Printf("err = %v\n", err)
		tx.Rollback()
		return
	}
	tx.Commit()
}

func transaction(tx *gorm.DB) error {

	if err := tx.Create(&User{ //ID: 20,
		Name: "hello-transaction3",
	}).Error; err != nil {
		return err
	}

	if err := tx.Create(&User{ //ID: 20,
		Name: "hello-transaction4",
	}).Error; err != nil {
		return err
	}

	if err := tx.Create(&User{
		ID:   20,
		Name: "hello-transaction5",
	}).Error; err != nil {
		return err
	}

	return nil
}
