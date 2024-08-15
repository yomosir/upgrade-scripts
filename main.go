package main

import (
	"bufio"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// ServerConfig 定义了每个服务器的配置结构
type ServerConfig struct {
	ServerIP   string `yaml:"server-ip"`
	ServerPort int    `yaml:"server-port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
}

// Config 定义了整个 YAML 文件的结构
type Config struct {
	Server []ServerConfig `yaml:"server"`
}

type UpgradeInfo struct {
	imageName string
	imageTag  string
}

func main() {
	// 读取镜像列表文件
	readUpgradeList()
	// 读取服务列表
	readServerList()
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
		oneUpradeImage := UpgradeInfo{
			imageName: imageInfoArr[0],
			imageTag:  imageInfoArr[1],
		}
		fmt.Printf("Upgrade info, imageName : %s, imageTag : %s \n", oneUpradeImage.imageName, oneUpradeImage.imageTag)
		upgradeMap[oneUpradeImage.imageName] = oneUpradeImage
	}
	return upgradeMap
}

// 读取集群服务器的list
func readServerList() []ServerConfig {
	// 读取文件内容
	data, err := ioutil.ReadFile("server.yaml")
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

// 执行升级，遍历服务器，读取每台服务器列表，与升级服务镜像列表比较，如果匹配，则进行镜像拉取，docker-compose执行up -d 升级
func doUpgrade(serverList []ServerConfig) {

}

// 先保存上个版本的环境配置文件，
func envConfGenerate() {

}
