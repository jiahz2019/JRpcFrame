package utils

import (
	"JRpcFrame/jlog"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
)


var ServerConf *ServerConfig
var DBConf *DBConfig

type ServerConfig struct {
	Host       string //当前服务器主机IP
	TcpPort    int    //当前服务器主机监听端口号
	ServerName string //当前服务器名称
	ServerId   string //服务器id
    
	Master    		ClientConfig
	
	MaxConn          int    //当前服务器主机允许的最大链接个数
	ServerWorkerSize uint32 //业务工作Worker池的数量
	MaxWorkerTaskLen uint32 //业务工作Worker对应负责的任务队列最大任务存储数量
	MaxMsgChanLen    uint32 //SendBuffMsg发送消息的缓冲最大长度
	MaxPacketSize    uint32 //都需数据包的最大值

}

type DBConfig struct {
	User     string
	Password string
	Name     string
	Port     string
	IP       string
}


type ClientConfig struct {
	RemoteHost      string          //远程服务器主机ip
	RemoteTcpPort   int             //远程服务器主机端口号
	ClientName      string
	ClientId        string
}

//默认
func InitServer() {
	//初始化GlobalObject变量，设置一些默认值
	ServerConf = &ServerConfig{
		ServerName:       "HazoServer",
		TcpPort:          8080,
		Host:             "10.0.4.11",
		MaxConn:          12000,
		ServerWorkerSize: 10,
		MaxWorkerTaskLen: 1024,
		MaxMsgChanLen:    1024,
		MaxPacketSize:    4096,
	}
}

func InitDb() {
	//初始化GlobalObject变量，设置一些默认值
	DBConf = &DBConfig{
		User:     "root",
		Password: "jiahaozhou123",
		Name:     "hazo",
		Port:     "3306",
		IP:       "127.0.0.1",
	}
}


//读取用户的配置文件
func (s *ServerConfig) Load(configFile string) {

	if confFileExists, _ := PathExists(configFile); confFileExists != true {
		text := fmt.Sprintf("Config File %s is not exist!!", configFile)
		jlog.StdLogger.Error(text)
		panic(errors.New(text))
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	//将json数据解析到struct中
	err = json.Unmarshal(data, &s)
	if err != nil {
		jlog.StdLogger.Error("load config error")
		panic(err)
	}
    ServerConf=s
	jlog.StdLogger.Info("Config: ", s)

}