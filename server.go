package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
)

// execCmd 通过 SSH 客户端执行命令，并返回执行结果
//
// 参数：
// client *ssh.Client：SSH 客户端指针
// command string：需要执行的命令
//
// 返回值：
// string：命令执行结果
// error：执行命令过程中出现的错误，如果执行成功则为 nil
func execCmd(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		fmt.Errorf("无法创建 SSH 会话: %v", err)
	}
	defer session.Close()
	var output bytes.Buffer
	session.Stdout = &output
	fmt.Printf("执行命令 command : [%s] \n", command)
	if err := session.Run(command); err != nil {
		return "", fmt.Errorf("无法执行命令: %v", err)
	}
	return output.String(), nil
}
