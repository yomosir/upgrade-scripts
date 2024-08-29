package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
)

// 比较当前服务器与升级的信息中是否有重复的服务
// doCompare 函数通过SSH客户端连接到远程服务器，执行docker命令获取镜像列表，并根据传入的upgradeInfoMap进行过滤，返回满足条件的UpgradeInfo切片。
// 参数：
//
//	client *ssh.Client：SSH客户端实例
//	upgradeInfoMap map[string]UpgradeInfo：包含升级信息的映射，key为镜像名称，value为对应的UpgradeInfo结构体
//
// 返回值：
//
//	[]UpgradeInfo：满足条件的UpgradeInfo切片
func doCompare(applications []string, upgradeInfoMap map[string]UpgradeInfo) []UpgradeInfo {

	var result []UpgradeInfo
	for _, application := range applications {
		imageName := fmt.Sprintf("dockerhub.kubekey.local/energycloud/%s", application)
		fmt.Printf("当前镜像：%s ", imageName)
		if value, exists := upgradeInfoMap[imageName]; exists {
			fmt.Printf("需要执行升级，升级的版本：%s", value.imageTag)
			result = append(result, value)
		}
		fmt.Print("\n")
	}
	return result
}

func loadImage(filename string, config ServerConfig) {
	if filename == "" {
		fmt.Println("请输入完整文件名")
	} else {
		if config.ImageLoad != "y" {
			fmt.Println("没有配置镜像加载服务器")
			return
		}
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
		// load 镜像文件到服务器上
		fmt.Printf(">>>>>>>>>> 开始加载镜像 <<<<<<<<<<\n")
		loadCmd := fmt.Sprintf("docker load -i %s", filename)
		_, err = execCmd(client, loadCmd)
		if err != nil {
			fmt.Printf("load image error: %v \n", err)
		}
		fmt.Printf(">>>>>>>>>> 镜像加载完成 <<<<<<<<<<\n")
	}

}

func pushImage(upgradeInfo map[string]UpgradeInfo, config ServerConfig) {
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
	// 推送镜像到harbor仓库中
	fmt.Printf(">>>>>>>>>> 开始推送镜像 <<<<<<<<<<\n")
	for _, value := range upgradeInfo {
		fullImageInfo := fmt.Sprintf("%s:%s", value.imageName, value.imageTag)
		pushCmd := fmt.Sprintf("docker push %s", fullImageInfo)
		_, err = execCmd(client, pushCmd)
		if err != nil {
			fmt.Printf("push image error: %v \n", err)
		}
	}
	fmt.Printf(">>>>>>>>>> 镜像推送完成 <<<<<<<<<<\n")
}
