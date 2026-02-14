// Package main 数据库迁移工具入口
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/TGlimmer/TG_walletbot/internal/domain/entity"
	"github.com/TGlimmer/TG_walletbot/internal/infrastructure/config"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "create-admin" {
		runCreateAdmin(os.Args[2:])
		return
	}

	runMigrate()
}

// runMigrate 执行数据库迁移
func runMigrate() {
	// 命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	direction := flag.String("direction", "up", "迁移方向: up / down")
	migrationsPath := flag.String("path", "file://migrations", "迁移文件路径")
	flag.Parse()

	// 位置参数: go run ./cmd/migrate up
	args := flag.Args()
	if len(args) > 0 {
		*direction = args[0]
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 构造 MySQL DSN (golang-migrate 格式)
	dsn := fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&multiStatements=true",
		cfg.Database.User, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Name, cfg.Database.Charset,
	)
	if cfg.Database.Charset == "" {
		dsn = fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?parseTime=True&loc=Local&multiStatements=true",
			cfg.Database.User, cfg.Database.Password,
			cfg.Database.Host, cfg.Database.Port,
			cfg.Database.Name,
		)
	}

	m, err := migrate.New(*migrationsPath, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化迁移失败: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	fmt.Printf("数据库迁移 [%s]\n", *direction)

	switch *direction {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	default:
		fmt.Fprintf(os.Stderr, "无效的迁移方向: %s (支持 up / down)\n", *direction)
		os.Exit(1)
	}

	if err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "迁移失败: %v\n", err)
		os.Exit(1)
	}

	if err == migrate.ErrNoChange {
		fmt.Println("无需迁移，数据库已是最新")
	} else {
		fmt.Println("迁移完成")
	}
}

// runCreateAdmin 创建管理员账号
// 用法: go run cmd/migrate/main.go create-admin --username=admin --password=mypassword
func runCreateAdmin(args []string) {
	fs := flag.NewFlagSet("create-admin", flag.ExitOnError)
	username := fs.String("username", "", "管理员用户名 (必填)")
	password := fs.String("password", "", "管理员密码 (必填)")
	configPath := fs.String("config", "", "配置文件路径")
	role := fs.String("role", "super_admin", "管理员角色 (super_admin / admin / viewer)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "解析参数失败: %v\n", err)
		os.Exit(1)
	}

	// 参数验证
	if *username == "" {
		fmt.Fprintln(os.Stderr, "错误: --username 不能为空")
		fs.Usage()
		os.Exit(1)
	}
	if *password == "" {
		fmt.Fprintln(os.Stderr, "错误: --password 不能为空")
		fs.Usage()
		os.Exit(1)
	}
	if *role != entity.AdminRoleSuperAdmin && *role != entity.AdminRoleAdmin && *role != entity.AdminRoleViewer {
		fmt.Fprintf(os.Stderr, "错误: --role 必须是 super_admin / admin / viewer 之一\n")
		os.Exit(1)
	}

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(cfg.Database.DSN()), &gorm.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接数据库失败: %v\n", err)
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// bcrypt 哈希密码
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "密码哈希生成失败: %v\n", err)
		os.Exit(1)
	}

	// 检查用户名是否已存在
	var existing entity.AdminUser
	result := db.Where("username = ?", *username).First(&existing)
	if result.Error == nil {
		fmt.Fprintf(os.Stderr, "错误: 用户名 '%s' 已存在 (ID=%d, 角色=%s)\n", *username, existing.ID, existing.Role)
		os.Exit(1)
	}

	// 创建管理员
	admin := entity.AdminUser{
		Username:     *username,
		PasswordHash: string(hash),
		Role:         *role,
		Status:       1,
	}
	if err := db.Create(&admin).Error; err != nil {
		fmt.Fprintf(os.Stderr, "创建管理员失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("管理员创建成功\n")
	fmt.Printf("  用户名: %s\n", admin.Username)
	fmt.Printf("  角色:   %s\n", admin.Role)
	fmt.Printf("  ID:     %d\n", admin.ID)
}
