package ldappool

import (
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// LdapConnCfg LDAP服务器连接配置
type ldapCfg struct {
	// 连接地址
	ConnUrl string `json:"conn_url" gorm:"type:varchar(255);unique_index;not null;comment:连接地址 逻辑外键"`
	// SSL加密方式
	SslEncryption bool `json:"ssl_encryption" gorm:"type:tinyint;length:1;comment:SSL加密方式"`
	// 超时设置
	Timeout time.Duration `json:"timeout" gorm:"type:int;comment:超时设置"`
	// 根目录
	BaseDn string `json:"base_dn" gorm:"type:varchar(255);not null;comment:根目录"`
	// 用户名
	AdminAccount string `json:"admin_account" gorm:"type:varchar(255);not null;comment:用户名"`
	// 密码
	Password string `json:"password" gorm:"type:varchar(255);not null;comment:密码"`
}

// 测试时给定的LDAP连接配置
var cfg = ldapCfg{
	ConnUrl:       "ldap://xx.xx.xx.xx:389",
	SslEncryption: false,
	Timeout:       5,
	BaseDn:        "DC=XXX,DC=com",
	AdminAccount:  "CN=Admin,CN=Users,DC=XXX,DC=com",
	Password:      "XXXXXXXXX",
}

// 需要返回的用户的属性列表
var attrs = []string{
	"employeeNumber",     // 工号
	"sAMAccountName",     // SAM账号
	"distinguishedName",  // dn
	"UserAccountControl", // 用户账户控制
	"accountExpires",     // 账户过期时间
	"pwdLastSet",         // 用户下次登录必须修改密码
	"whenCreated",        // 创建时间
	"whenChanged",        // 修改时间
	"displayName",        // 显示名
	"sn",                 // 姓
	"name",
	"givenName",  // 名
	"mail",       // 邮箱
	"mobile",     // 手机号
	"company",    // 公司
	"department", // 部门
	"title",      // 职务
	"cn",         // common name
}

// 测试连接池
func TestLdapPool(t *testing.T) {
	pool, err := NewChannelPool(10, 1000, "testLdapPool",
		func(name string) (ldap.Client, error) {
			conn, err := ldap.DialURL(cfg.ConnUrl)
			if err != nil {
				fmt.Println("Fail to dial ldap url, err: ", err)
			}

			// 重新连接TLS
			if err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
				fmt.Println("Fail to start tls, err: ", err)
			}

			// 与只读用户绑定
			if err = conn.Bind(cfg.AdminAccount, cfg.Password); err != nil {
				fmt.Println("admin user auth failed, err: ", err)
			}
			return conn, nil
		}, []uint16{ldap.LDAPResultTimeLimitExceeded, ldap.ErrorNetwork})
	if err != nil {
		fmt.Println(err)
	}
	defer pool.Close()

	fmt.Println(pool.Len())

	var poolConn *PoolConn
	for i := 0; i < 13; i++ {
		poolConn, err = pool.Get()
		if err != nil {
			fmt.Println(err)
		}
		// defer
		// poolConn.Close()
		fmt.Println(poolConn)
		fmt.Println(pool.Len())
	}
	fmt.Println(poolConn)
	fmt.Println(pool.Len())
}

// 测试查询用户
func TestFetchUser(t *testing.T) {
	pool, err := NewChannelPool(5, 100, "testFetchUserLdapPool",
		func(name string) (ldap.Client, error) {
			conn, err := ldap.DialURL(cfg.ConnUrl)
			if err != nil {
				fmt.Println("Fail to dial ldap url, err: ", err)
			}

			// 重新连接TLS
			if err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
				fmt.Println("Fail to start tls, err: ", err)
			}

			// 与只读用户绑定
			if err = conn.Bind(cfg.AdminAccount, cfg.Password); err != nil {
				fmt.Println("admin user auth failed, err: ", err)
			}
			return conn, nil
		}, []uint16{ldap.LDAPResultTimeLimitExceeded, ldap.ErrorNetwork})
	if err != nil {
		fmt.Println(err)
	}
	conn, _ := pool.Get()

	// 多查询条件 根据employeeNumber和displayName字段查询
	ldapFilterNum := "(employeeNumber=" + "9527" + ")"
	ldapFilterName := "(displayName=" + "张三" + ")"
	searchFilter := "(&(objectClass=user)(mail=*))" // 有邮箱的用户 排除系统级别用户
	searchFilter += ldapFilterNum
	searchFilter += ldapFilterName
	searchFilter = "(&" + searchFilter + ")"

	searchRequest := ldap.NewSearchRequest(
		cfg.BaseDn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 500, 0, false,
		searchFilter,
		attrs,
		nil,
	)
	// 分页查询
	sr, err := conn.SearchWithPaging(searchRequest, 100)
	if err != nil {
		fmt.Println("Fail to search users, err: ", err)
	}
	if len(sr.Entries) > 0 && len(sr.Entries[0].Attributes) > 0 {
		result := sr.Entries
		fmt.Println(result[0])
	}
}
