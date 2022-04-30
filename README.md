# github.com/randolphcyg/ldappool

Based on ldapV3 ldap connection pool, it is convenient to quickly obtain ldap connection objects.

## 1. install

```shell
go get github.com/randolphcyg/ldappool
```

## 2. Usage

This case is to initialize the ldap connection pool, query all users and store the dn attribute in the file.

```go
package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/randolphcyg/ldappool"
	"github.com/go-ldap/ldap/v3"
	"os"
	"time"
)

var (
	LdapConns LdapConn
	LdapPool  ldappool.Pool
	attrs = []string{
		"employeeNumber",     // 工号
		"sAMAccountName",     // SAM账号
		"distinguishedName",  // dn
		"userAccountControl", // 用户账户控制
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
	}
)


// LdapConn LDAP服务器连接配置
type LdapConn struct {
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

type LdapAttributes struct {
	// ldap字段
	Num         string `json:"employeeNumber" gorm:"type:varchar(100);unique_index"`     // 工号
	Sam         string `json:"sAMAccountName" gorm:"type:varchar(128);unique_index"`     // SAM账号
	Dn          string `json:"distinguishedName" gorm:"type:varchar(100);unique_index"`  // dn
	AccountCtl  string `json:"UserAccountControl" gorm:"type:varchar(100);unique_index"` // 用户账户控制
	Expire      string `json:"accountExpires" gorm:"type:varchar(100);unique_index"`     //  账户过期时间
	PwdLastSet  string `json:"pwdLastSet" gorm:"type:varchar(100);unique_index"`         //  用户下次登录必须修改密码
	WhenCreated string `json:"whenCreated" gorm:"type:varchar(100);unique_index"`        //  创建时间
	WhenChanged string `json:"whenChanged" gorm:"type:varchar(100);unique_index"`        //  修改时间
	DisplayName string `json:"displayName" gorm:"type:varchar(32);unique_index"`         //  真实姓名
	Sn          string `json:"sn" gorm:"type:varchar(100);unique_index"`                 //  姓
	Name        string `json:"name" gorm:"type:varchar(100);unique_index"`               // 姓名
	GivenName   string `json:"givenName" gorm:"type:varchar(100);unique_index"`          // 名
	Email       string `json:"mail" gorm:"type:varchar(128);unique_index"`               // 邮箱
	Phone       string `json:"mobile" gorm:"type:varchar(32);unique_index"`              // 移动电话
	Company     string `json:"company" gorm:"type:varchar(128);unique_index"`            // 公司
	Depart      string `json:"department" gorm:"type:varchar(128);unique_index"`         // 部门
	Title       string `json:"title" gorm:"type:varchar(100);unique_index"`              // 职务
}

// Init 初始化ldap连接池
func Init(c *LdapConn) (err error) {
	LdapConns = LdapConn{
		ConnUrl:       c.ConnUrl,
		SslEncryption: c.SslEncryption,
		Timeout:       c.Timeout,
		BaseDn:        c.BaseDn,
		AdminAccount:  c.AdminAccount,
		Password:      c.Password,
	}
	// 初始化ldap连接池 初始连接数是50
	LdapPool, err = ldappool.NewChannelPool(50, 1000, "originalLdapPool",
		func(s string) (ldap.Client, error) {
			conn, err := ldap.DialURL(LdapConns.ConnUrl)
			if err != nil {
				fmt.Print("Fail to dial ldap url, err: ", err)
			}

			// 重新连接TLS
			if err = conn.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
				fmt.Print("Fail to start tls, err: ", err)
			}

			// 与只读用户绑定
			if err = conn.Bind(LdapConns.AdminAccount, LdapConns.Password); err != nil {
				fmt.Print("admin user auth failed, err: ", err)
			}
			return conn, nil
		}, []uint16{ldap.LDAPResultTimeLimitExceeded, ldap.ErrorNetwork})
	if err != nil {
		fmt.Print(err)
		return
	}
	return
}

func NewLdapConnContext() *LdapConn {
	return &LdapConn{}
}

func FetchLdapUsers(user *LdapAttributes) (result []*ldap.Entry) {
	// 获取连接
	LdapConn, err := LdapPool.Get()
	if err != nil {
		fmt.Printf(err.Error())
	}

	// 多查询条件
	ldapFilterNum := "(employeeNumber=" + user.Num + ")"
	ldapFilterSam := "(sAMAccountName=" + user.Sam + ")"
	ldapFilterEmail := "(mail=" + user.Email + ")"
	ldapFilterPhone := "(mobile=" + user.Phone + ")"
	ldapFilterName := "(displayName=" + user.DisplayName + ")"
	ldapFilterDepart := "(department=" + user.Depart + ")"
	ldapFilterCompany := "(company=" + user.Company + ")"
	ldapFilterTitle := "(title=" + user.Title + ")"

	searchFilter := "(&(objectClass=user)(mail=*))" // 有邮箱的用户 排除系统级别用户

	if user.Num != "" {
		searchFilter += ldapFilterNum
	}
	if user.Sam != "" {
		searchFilter += ldapFilterSam
	}
	if user.Email != "" {
		searchFilter += ldapFilterEmail
	}
	if user.Phone != "" {
		searchFilter += ldapFilterPhone
	}
	if user.DisplayName != "" {
		searchFilter += ldapFilterName
	}
	if user.Depart != "" {
		searchFilter += ldapFilterDepart
	}
	if user.Company != "" {
		searchFilter += ldapFilterCompany
	}
	if user.Title != "" {
		searchFilter += ldapFilterTitle
	}
	searchFilter = "(&" + searchFilter + ")"

	searchRequest := ldap.NewSearchRequest(
		LdapConns.BaseDn,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 500, 0, false,
		searchFilter,
		attrs,
		nil,
	)

	sr, err := LdapConn.SearchWithPaging(searchRequest, 100)
	if err != nil {
		fmt.Printf("Fail to search users, err: " + err.Error())
	}
	if len(sr.Entries) > 0 && len(sr.Entries[0].Attributes) > 0 {
		result = sr.Entries
	}
	return
}

func main() {
	ldapConnContext := NewLdapConnContext()
	flag.StringVar(&ldapConnContext.ConnUrl, "connUrl", "ldap://192.168.x.x:389", "ad域连接地址")
	flag.StringVar(&ldapConnContext.BaseDn, "baseDn", "OU=组织名称,DC=XXX,DC=com", "基础dn")
	flag.StringVar(&ldapConnContext.AdminAccount, "adminAccount", "CN=Admin用户名,CN=Users,DC=XXX,DC=com", "管理员用户")
	flag.StringVar(&ldapConnContext.Password, "password", "Admin用户密码", "管理员用户密码")
	flag.BoolVar(&ldapConnContext.SslEncryption, "sslEncryption", false, "是否ssl")
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Parse()

	err := Init(ldapConnContext)
	if err != nil {
		fmt.Printf(err.Error())
	}

	filePath := "./allUserDns.txt"
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("文件打开失败", err)
	}
	// 及时关闭file句柄
	defer file.Close()
	// 写入文件时，使用带缓存的 *Writer
	write := bufio.NewWriter(file)

	users := FetchLdapUsers(&LdapAttributes{})
	for idx, user := range users {
		fmt.Println(idx+1, user.DN)
		//break
		write.WriteString(user.DN + "\n")
		break
	}

	//Flush将缓存的文件真正写入到文件中
	write.Flush()

}
```
