package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
)

// 执行升级，遍历服务器，读取每台服务器列表，与升级服务镜像列表比较，如果匹配，则进行镜像拉取，docker-compose执行up -d 升级
func doUpgrade(serverList []ServerConfig, upgradeInfoMap map[string]UpgradeInfo) {
	fmt.Printf(">>>>>>>>>> 开始升级 <<<<<<<<<<\n")
	for _, config := range serverList {
		doServerProcess(config, upgradeInfoMap)
		fmt.Printf(">>>>>>>>>> %s 升级完成 <<<<<<<<<<\n", config.ServerIP)
	}
}

func doServerProcess(config ServerConfig, upgradeInfoMap map[string]UpgradeInfo) {
	// 配置 SSH 客户端
	clientConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接到 SSH 服务器
	client, err := ssh.Dial("tcp", config.ServerIP+":"+config.ServerPort, clientConfig)
	if err != nil {
		fmt.Errorf("无法连接到 SSH 服务器: %v", err)
	}
	defer client.Close()
	// 将服务器上当前的运行容器的镜像与升级列表中进行对比，如果有的话则记录下来，后面进行升级
	fmt.Println("开始对比镜像列表")
	matchedUpdateInfo := doCompare(config.applications, upgradeInfoMap)
	if nil == matchedUpdateInfo || 0 == len(matchedUpdateInfo) {
		// 如果匹配结果为空数组，则退出
		fmt.Println("当前服务器无需要升级节点跳过")
		return
	}
	fmt.Println("开始备份环境变量文件")
	// 修改环境变量文件
	err = envConfGenerate(client, matchedUpdateInfo)
	if err != nil {
		fmt.Errorf("修改环境变量文件失败: %v", err)
	} else {
		fmt.Println("修改环境变量文件成功")
	}
	// 通过docker-compose命令执行升级
	err = dockerComposeUpgrade(client, matchedUpdateInfo)
	if err != nil {
		fmt.Errorf("执行升级命令失败: %v", err)
	} else {
		fmt.Println("执行升级命令成功")
	}
}

func dockerComposeUpgrade(client *ssh.Client, upgradeInfo []UpgradeInfo) error {
	// 执行升级命令
	for _, info := range upgradeInfo {
		command := fmt.Sprintf("docker-compose -f /root/.docker/docker-compose.yml up -d --no-deps %s \n", info.serviceName)
		_, err := execCmd(client, command)
		if err != nil {
			return fmt.Errorf("执行升级命令失败: %v \n", err)
		} else {
			fmt.Printf("执行升级成功，命令：%s \n", command)
		}
	}
	return nil
}
