package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"cmsingbox.local/cmsingbox/internal/licensing"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen(os.Args[2:])
	case "issue":
		err = issue(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "用法: license-tool keygen -private license-private.key -public license-public.key")
	fmt.Fprintln(os.Stderr, "      license-tool issue -private license-private.key -device 123456 -subscriptions 10 [-expires 2027-12-31] [-id customer-001]")
	os.Exit(2)
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	privatePath := fs.String("private", "license-private.key", "私钥输出路径")
	publicPath := fs.String("public", "license-public.key", "公钥输出路径")
	if err := fs.Parse(args); err != nil {
		return err
	}
	publicKey, privateKey, err := licensing.GenerateKey()
	if err != nil {
		return err
	}
	if err := writeExclusive(*privatePath, []byte(base64.StdEncoding.EncodeToString(privateKey)+"\n"), 0600); err != nil {
		return fmt.Errorf("写入私钥失败: %w", err)
	}
	if err := writeExclusive(*publicPath, []byte(base64.StdEncoding.EncodeToString(publicKey)+"\n"), 0644); err != nil {
		return fmt.Errorf("写入公钥失败: %w", err)
	}
	fmt.Println("密钥已生成。私钥必须离线保管，不能放入仓库或服务器。")
	fmt.Println("公钥:", base64.StdEncoding.EncodeToString(publicKey))
	return nil
}

func issue(args []string) error {
	fs := flag.NewFlagSet("issue", flag.ContinueOnError)
	privatePath := fs.String("private", "license-private.key", "私钥文件")
	device := fs.String("device", "", "六位设备码")
	maxSubscriptions := fs.Int("subscriptions", 0, "允许的订阅链接数")
	expires := fs.String("expires", "", "到期日 YYYY-MM-DD；留空表示永久")
	licenseID := fs.String("id", "", "授权编号")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(*device) != 6 || *maxSubscriptions < 1 {
		return fmt.Errorf("device 必须为六位设备码，subscriptions 必须大于 0")
	}
	raw, err := os.ReadFile(*privatePath)
	if err != nil {
		return err
	}
	privateKey, err := licensing.ParsePrivateKey(string(raw))
	if err != nil {
		return err
	}
	var expiresAt int64
	if strings.TrimSpace(*expires) != "" {
		date, err := time.ParseInLocation("2006-01-02", *expires, time.Local)
		if err != nil {
			return fmt.Errorf("到期日格式应为 YYYY-MM-DD")
		}
		expiresAt = date.Add(24 * time.Hour).Unix()
	}
	id := strings.TrimSpace(*licenseID)
	if id == "" {
		id = fmt.Sprintf("LIC-%d", time.Now().Unix())
	}
	token, err := licensing.Issue(ed25519.PrivateKey(privateKey), licensing.Claims{
		LicenseID: id, DeviceCode: *device, MaxSubscriptions: *maxSubscriptions, ExpiresAt: expiresAt,
	})
	if err != nil {
		return err
	}
	fmt.Println(token)
	return nil
}

func writeExclusive(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
