package main

import (
	"bufio"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strings"
)

// ServerConfig 定义了每个服务器的配置结构
type ServerConfig struct {
	ServerIP     string   `yaml:"server-ip"`
	ServerPort   string   `yaml:"server-port"`
	Username     string   `yaml:"username"`
	Password     string   `yaml:"password"`
	ImageLoad    string   `yaml:"image-load"`
	applications []string `yaml:"applications"`
}

// Config 定义了整个 YAML 文件的结构
type Config struct {
	Server []ServerConfig `yaml:"server"`
}

type UpgradeInfo struct {
	imageName   string
	imageTag    string
	serviceName string
}

func main() {

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("请输入参数: upgrade(升级), load [文件名](加载&推送镜像)")
	}
	// 读取镜像列表文件
	upgradeMap := readUpgradeList()
	// 读取服务列表
	serverList := readServerList()
	var baseServerConfig = ServerConfig{}
	for _, serverConf := range serverList {
		if serverConf.ImageLoad == "y" {
			baseServerConfig = serverConf
		}
	}
	// 执行推送镜像操作
	if args[0] == "upgrade" {
		// 执行升级
		doUpgrade(serverList, upgradeMap)
	} else if args[0] == "load" {
		// 加载镜像
		loadImage(args[1], baseServerConfig)
	} else if args[0] == "push" {
		if baseServerConfig.ImageLoad != "y" {
			fmt.Println("没有配置镜像加载服务器")
		}
		pushImage(upgradeMap, baseServerConfig)
	} else if args[0] == "install" {
		// 执行安装服务操作，仅支持docker应用的安装，不支持基础服务组件安装
	} else if args[0] == "deploy" {
		// 执行部署操作
	}
	fmt.Println("操作完成")
}

// 读取升级镜像列表
func readUpgradeList() map[string]UpgradeInfo {
	file, err := os.Open("./upgrade.txt")
	if err != nil {
		_ = fmt.Errorf("read upgrade list error: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var upgradeMap = map[string]UpgradeInfo{}
	for scanner.Scan() {
		line := scanner.Text()
		imageInfoArr := strings.Split(line, ":")
		imageNameArr := strings.Split(imageInfoArr[0], "/")
		length := len(imageNameArr)
		oneUpgradeImage := UpgradeInfo{
			imageName:   imageInfoArr[0],
			imageTag:    imageInfoArr[1],
			serviceName: imageNameArr[length-1],
		}
		fmt.Printf("Upgrade info, imageName : %s, imageTag : %s \n", oneUpgradeImage.imageName, oneUpgradeImage.imageTag)
		upgradeMap[oneUpgradeImage.imageName] = oneUpgradeImage
	}
	return upgradeMap
}

// 读取集群服务器的list
func readServerList() []ServerConfig {
	// 读取文件内容
	data, err := os.ReadFile("server.yaml")
	if err != nil {
		log.Fatalf("无法读取文件: %v", err)
	}
	// 定义配置结构的实例
	var config Config
	// 解析 YAML 文件内容
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("无法解析 YAML 文件: %v", err)
	}
	// 输出读取的配置信息
	for _, server := range config.Server {
		fmt.Printf("Server IP: %s, Port: %d, Username: %s \n",
			server.ServerIP, server.ServerPort, server.Username)

	}
	return config.Server
}
