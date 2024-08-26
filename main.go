package main

import (
	"bufio"
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strings"
	"time"
)

// ServerConfig 定义了每个服务器的配置结构
type ServerConfig struct {
	ServerIP   string `yaml:"server-ip"`
	ServerPort string `yaml:"server-port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	ImageLoad  string `yaml:"image-load"`
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
	if args[0] == "upgrade" {
		// 执行升级
		doUpgrade(serverList, upgradeMap)
	} else if args[0] == "load" {
		fileName := args[1]
		if fileName == "" {
			fmt.Println("请输入完整文件名")
		} else {
			var config = ServerConfig{}
			for _, serverConf := range serverList {
				if serverConf.ImageLoad == "y" {
					config = serverConf
				}
			}
			if config.ImageLoad != "y" {
				fmt.Println("没有配置镜像加载服务器")
			}
			// 加载镜像
			loadImage(fileName, config, upgradeMap)
		}
	}

}

func loadImage(filename string, config ServerConfig, upgradeInfo map[string]UpgradeInfo) {
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

// 执行升级，遍历服务器，读取每台服务器列表，与升级服务镜像列表比较，如果匹配，则进行镜像拉取，docker-compose执行up -d 升级
func doUpgrade(serverList []ServerConfig, upgradeInfoMap map[string]UpgradeInfo) {
	fmt.Printf(">>>>>>>>>> 开始升级 <<<<<<<<<<\n")
	for _, config := range serverList {
		doServerProcess(config, upgradeInfoMap)
		fmt.Printf(">>>>>>>>>> %s 升级完成 <<<<<<<<<<\n", config.ServerIP)
	}
}

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
	matchedUpdateInfo := doCompare(client, upgradeInfoMap)
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
func doCompare(client *ssh.Client, upgradeInfoMap map[string]UpgradeInfo) []UpgradeInfo {
	session, err := client.NewSession()
	if err != nil {
		fmt.Errorf("无法创建 SSH 会话: %v", err)
		return nil
	}
	defer session.Close()
	var output bytes.Buffer
	session.Stdout = &output
	if err := session.Run("docker ps -a --format {{.Image}}"); err != nil {
		fmt.Errorf("执行命令失败：%v", err)
	}
	images := strings.Split(strings.TrimSpace(output.String()), "\n")
	var result []UpgradeInfo
	for _, image := range images {
		imageInfoArr := strings.Split(image, ":")
		fmt.Printf("当前镜像及版本：%s ", image)
		if value, exists := upgradeInfoMap[imageInfoArr[0]]; exists {
			fmt.Printf("需要执行升级，升级的版本：%s", value.imageTag)
			result = append(result, value)
		}
		fmt.Print("\n")
	}
	return result
}
