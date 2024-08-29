package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"strings"
	"time"
)

// 先保存上个版本的环境配置文件，
// envConfGenerate 函数用于生成环境变量配置文件
//
// 参数：
// client *ssh.Client：SSH客户端指针，用于执行远程命令
// upgradeInfo []UpgradeInfo：升级信息切片，包含镜像名称和标签等信息
//
// 返回值：
// error：如果备份环境变量文件失败或写入环境变量文件失败，则返回错误信息；否则返回nil
func envConfGenerate(client *ssh.Client, upgradeInfo []UpgradeInfo) error {
	// 执行环境变量文件的备份
	err := backupEnvFile(client)
	if err != nil {
		return fmt.Errorf("backup env file error: %v", err)
	}
	fmt.Println("备份环境变量文件成功, 开始写入新环境变量文件.....")
	// 生成新的.env文件
	filePath := "/root/.docker/.env"
	readCmd := fmt.Sprintf("cat %s", filePath)
	content, err := execCmd(client, readCmd)
	paramVersionMap := make(map[string]string)
	for _, oneInfo := range upgradeInfo {
		imageNameInfoArr := strings.Split(oneInfo.imageName, "/")
		serverName := imageNameInfoArr[len(imageNameInfoArr)-1]
		paramName := fmt.Sprintf("%s_version", strings.ReplaceAll(serverName, "-", "_"))
		fmt.Printf("paramName: %s, imageTag: %s \n", paramName, oneInfo.imageTag)
		paramVersionMap[paramName] = oneInfo.imageTag
	}
	newContent := doParamUpdate(client, paramVersionMap, content)
	writeCmd := fmt.Sprintf("echo '%s' > %s", newContent, filePath)
	_, err = execCmd(client, writeCmd)
	if err != nil {
		return fmt.Errorf("write env file error: %v", err)
	}
	return nil
}

// doParamUpdate 函数用于执行环境变量参数的替换操作
//
// 参数：
// client *ssh.Client：SSH客户端指针，用于SSH连接，但在此函数中并未使用
// paramVersionMap map[string]string：参数版本映射，键为参数名，值为参数的新版本或新值
// content string：待替换参数的文本内容
//
// 返回值：
// string：替换后的文本内容
func doParamUpdate(client *ssh.Client, paramVersionMap map[string]string, content string) string {
	// 执行环境变量参数替换
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lineInfo := strings.Split(line, "=")
		value, exists := paramVersionMap[lineInfo[0]]
		// 如果更新的镜像中存在，那么则变更数组中该条配置
		if exists {
			lines[i] = fmt.Sprintf("%s=%s", lineInfo[0], value)
			fmt.Printf("参数 %s 已更新为 %s, 该行最新为: %s \n", lineInfo[0], value, lines[i])
		}
	}
	return strings.Join(lines, "\n")
}

func backupEnvFile(client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		fmt.Errorf("无法创建 SSH 会话: %v", err)
	}
	defer session.Close()
	now := time.Now()
	dir := fmt.Sprintf("/root/.docker/version/%d-%02d-%02d", now.Year(), now.Month(), now.Day())
	result, err := checkIfDirExist(client, dir)
	if err != nil {
		fmt.Errorf("执行判断环境变量失败: %v", err)
		return nil
	}
	if "exists\n" == result {
		fmt.Printf("备份文件夹已经存在，%s \n", dir)
		return nil
	}
	// 备份
	command := fmt.Sprintf("mkdir -p %s&&cp /root/.docker/.env %s&&cp /root/.docker/docker-compose.yml %s", dir, dir, dir)
	err = session.Run(command)
	if err != nil {
		return fmt.Errorf("执行备份命令失败, %v", err)
	} else {
		fmt.Printf("执行备份成功.....，命令：%s \n", command)
	}
	return nil
}

// checkIfDirExist 检查SSH服务器上指定路径是否为目录
//
// 参数:
//
//	client: *ssh.Client类型，SSH客户端实例
//	path: string类型，待检查的路径
//
// 返回值:
//
//	string类型，若路径存在则返回"exists"，否则返回"not exists"
//	error类型，如果执行过程中出现错误则返回错误信息，否则返回nil
func checkIfDirExist(client *ssh.Client, path string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		fmt.Errorf("无法创建 SSH 会话: %v", err)
	}
	defer session.Close()
	command := fmt.Sprintf("if [ -d \"%s\" ]; then echo \"exists\"; else echo \"not exists\"; fi", path)
	// 执行命令
	var output bytes.Buffer
	session.Stdout = &output
	if err := session.Run(command); err != nil {
		return "", fmt.Errorf("无法执行命令: %v", err)
	}
	return output.String(), nil
}
