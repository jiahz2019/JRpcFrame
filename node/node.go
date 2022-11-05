package node

import (
	"JRpcFrame/cluster"
	"JRpcFrame/console"
	"JRpcFrame/jlog"
	"JRpcFrame/jtimer"
	"JRpcFrame/profiler"
	"JRpcFrame/service"
	"JRpcFrame/utils/buildtime"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var nodeLogger *jlog.Logger          //节点日志器
var nodeTimer *jtimer.TimerScheduler //节点时间轮定时器
var node *Node

//是否开启了控制台
var isOpenConsole bool

type Node struct {
	nodeId    int       //节点id
	closeSig  chan bool //关闭信号
	sig       chan os.Signal
	configDir string //节点配置的文件夹

	logLevel string //日志等级
	logPath  string //日志输出路径

	bValid           bool               //节点是否有效
	preSetupService  []service.IService //将要安装的所有service
	profilerInterval time.Duration      //性能监测的间隔时间
}

func init() {
	node = NewNode()
	signal.Notify(node.sig, syscall.SIGINT, syscall.SIGTERM, syscall.Signal(10))
	//为节点的控制台注册默认命令
	console.RegisterCommandBool("help", false, "<-help> This help.", usage)
	console.RegisterCommandString("name", "", "<-name nodeName> Node's name.", node.setName)
	console.RegisterCommandString("start", "", "<-start nodeid=nodeid> Run server.", node.startNode)
	console.RegisterCommandString("stop", "", "<-stop nodeid=nodeid> Stop server process.", node.stopNode)
	console.RegisterCommandString("config", "", "<-config path> Configuration file path.", setConfigPath)
	console.RegisterCommandString("console", "", "<-console true|false> Turn on or off screen log output.", openConsole)
	console.RegisterCommandString("loglevel", "debug", "<-loglevel debug|release|warning|error|fatal> Set loglevel.", setLogLevel)
	console.RegisterCommandString("logpath", "", "<-logpath path> Set log file path.", setLogPath)
	console.RegisterCommandString("pprof", "", "<-pprof ip:port> Open performance analysis.", setPprof)
}

/*
   @brief: 生成一个节点
   @param [in] nodeId: 节点id
*/
func NewNode() *Node {
	node := &Node{
		sig:              make(chan os.Signal, 3),
		closeSig:         make(chan bool, 1),
		profilerInterval: 0,
	}
	return node
}

func (n *Node) startNode(args interface{}) error {
	//1.解析参数
	param := args.(string)
	if param == "" {
		return nil
	}

	sParam := strings.Split(param, "=")
	if len(sParam) != 2 {
		return fmt.Errorf("invalid option %s", param)
	}
	if sParam[0] != "nodeid" {
		return fmt.Errorf("invalid option %s", param)
	}
	nodeId, err := strconv.Atoi(sParam[1])
	if err != nil {
		return fmt.Errorf("invalid option %s", param)
	}
	jlog.StdLogger.Info("Node start running server")

	//2.初始化node
	n.InitNode(nodeId)
	//3.运行service
	service.GetServiceMgr().StartAll()

	//4.运行集群
	cluster.GetCluster().Start()

	//5.记录进程id号
	n.writeProcessPid(nodeId)

	//6.监听程序退出信号&性能报告
	bRun := true
	var pProfilerTicker *time.Ticker = &time.Ticker{}
	if n.profilerInterval > 0 {
		pProfilerTicker = time.NewTicker(n.profilerInterval)
	}
	for bRun {
		select {
		case <-n.sig:
			jlog.StdLogger.Info("receipt stop signal.")
			bRun = false
		case <-pProfilerTicker.C:
			profiler.Report()
		}
	}
	cluster.GetCluster().Stop()
	//7.退出
	close(n.closeSig)
	service.GetServiceMgr().WaitStopAll()

	jlog.StdLogger.Info("Server is stop.")
	return nil

}

func (n *Node) stopNode(args interface{}) error {
	//解析参数
	param := args.(string)
	if param == "" {
		return nil
	}

	sParam := strings.Split(param, "=")
	if len(sParam) != 2 {
		return fmt.Errorf("invalid option %s", param)
	}
	if sParam[0] != "nodeid" {
		return fmt.Errorf("invalid option %s", param)
	}
	nodeId, err := strconv.Atoi(sParam[1])
	if err != nil {
		return fmt.Errorf("invalid option %s", param)
	}

	processId, err := n.getRunProcessPid(nodeId)
	if err != nil {
		return err
	}

	KillProcess(processId)
	return nil
}

/*
   @brief:节点开始运行
*/
func Start() {
	err := console.Run(os.Args)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
}

func (node *Node) InitNode(id int) {
	//1.初始化集群
	node.nodeId = id
	err := cluster.GetCluster().InitCluster(id, node.Setup)

	if err != nil {
		jlog.StdLogger.Fatal("read system config is error ", err.Error())
	}
	//初始化定时器
	nodeTimer = jtimer.NewAutoTimerScheduler()
	//初始化日志
	err = node.InitLog()
	if err != nil {
		return
	}

	//2.setup service
	for _, s := range node.preSetupService {
		//是否配置的service
		if cluster.GetCluster().IsConfigService(s.GetName()) == false {
			continue
		}

		pServiceCfg := cluster.GetCluster().GetServiceCfg(s.GetName())
		s.InitService(s, cluster.GetRpcClient, cluster.GetRpcServer, pServiceCfg)
	}

	//3.service初始化
	service.GetServiceMgr().InitServiceMgr(node.closeSig)
}

func (node *Node) InitLog() error {
	//默认日志路径
	if node.logPath == "" {
		setLogPath("./log")
	}

	localnodeinfo := cluster.GetCluster().GetLocalNodeInfo()
	prefix := fmt.Sprintf("%s_%d_log", localnodeinfo.NodeName, localnodeinfo.NodeId)
	nodeLogger = jlog.NewLogger(os.Stdout, prefix, jlog.DefaultFlag, jlog.LogDebug)
	filename := fmt.Sprintf("%s_%d_log", localnodeinfo.NodeName, localnodeinfo.NodeId)
	err := nodeLogger.SetLogFile(node.logPath, filename)
	if err != nil {
		fmt.Printf("cannot create log file!\n")
		return err
	}
	return nil
}

/*
    @brief:获取当前节点进程id
	@param [in] nodeId:需要获取进程id的节点
	@return: 当前节点进程id
*/
func (n *Node) getRunProcessPid(nodeId int) (int, error) {
	//从保存进程信息文件中读取进程id
	f, err := os.OpenFile(fmt.Sprintf("%s_%d.pid", os.Args[0], nodeId), os.O_RDONLY, 0600)
	defer f.Close()
	if err != nil {
		return 0, err
	}

	pidByte, errs := io.ReadAll(f)
	if errs != nil {
		return 0, errs
	}

	return strconv.Atoi(string(pidByte))
}

/*
    @brief:写入进程id
	@param [in] nodeId:需要写入进程id的节点
*/
func (n *Node) writeProcessPid(nodeId int) {
	//从保存进程信息文件中写入进程id
	f, err := os.OpenFile(fmt.Sprintf("%s_%d.pid", os.Args[0], nodeId), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0600)
	defer f.Close()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	} else {
		_, err = f.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(-1)
		}
	}
}

/*
   @brief: 输出所有的默认命令的备注
*/
func usage(val interface{}) error {
	ret := val.(bool)
	if ret == false {
		return nil
	}
	//输出编译时间信息
	if len(buildtime.GetBuildDateTime()) > 0 {
		fmt.Fprintf(os.Stderr, "Welcome to JRpcFrame(build info: %s)\nUsage: JRpcFrame [-help] [-start node=1] [-stop] [-config path] [-pprof 0.0.0.0:6060]...\n", buildtime.GetBuildDateTime())
	} else {
		fmt.Fprintf(os.Stderr, "Welcome to JRpcFrame\nUsage: JRpcFrame [-help] [-start node=1] [-stop] [-config path] [-pprof 0.0.0.0:6060]...\n")
	}
	//输出所有的默认命令
	console.PrintDefaults()
	return nil
}

/*
   @brief: 设置命令的名字
*/
func (n *Node) setName(val interface{}) error {
	return nil
}

/*
   @brief: 设置节点配置路径
*/
func setConfigPath(val interface{}) error {
	configPath := val.(string)
	if configPath == "" {
		return nil
	}
	_, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("Cannot find file path %s", configPath)
	}
	cluster.SetConfigDir(configPath)
	node.configDir = configPath
	return nil
}

/*
   @brief: 设置是否开起了控制台
*/
func openConsole(args interface{}) error {
	if args == "" {
		return nil
	}
	strOpen := strings.ToLower(strings.TrimSpace(args.(string)))
	if strOpen == "false" {
		isOpenConsole = false
	} else if strOpen == "true" {
		isOpenConsole = true
	} else {
		return errors.New("Parameter console error!")
	}
	return nil
}

/*
   @brief: 设置日志等级
*/
func setLogLevel(args interface{}) error {
	if args == "" {
		return nil
	}

	node.logLevel = strings.TrimSpace(args.(string))
	if node.logLevel != "debug" && node.logLevel != "info" && node.logLevel != "warn" && node.logLevel != "error" && node.logLevel != "panic" && node.logLevel != "fatal" {
		return errors.New("unknown level: " + node.logLevel)
	}
	return nil
}

/*
   @brief: 设置日志输出路径
*/
func setLogPath(args interface{}) error {
	if args == "" {
		return nil
	}
	node.logPath = strings.TrimSpace(args.(string))
	dir, err := os.Stat(node.logPath) //这个文件夹不存在
	if err == nil && dir.IsDir() == false {
		return errors.New("Not found dir " + node.logPath)
	}

	if err != nil {
		err = os.Mkdir(node.logPath, os.ModePerm)
		if err != nil {
			return errors.New("Cannot create dir " + node.logPath)
		}
	}

	return nil
}

func setPprof(val interface{}) error {
	listenAddr := val.(string)
	if listenAddr == "" {
		return nil
	}

	go func() {
		err := http.ListenAndServe(listenAddr, nil)
		if err != nil {
			panic(fmt.Errorf("%+v", err))
		}
	}()

	return nil
}

/*
   @brief:获取节点配置
*/
func (node *Node) GetConfigDir() string {
	return node.configDir
}

/*
   @brief:返回全局节点
*/
func GetNode() *Node {
	return node
}

/*
    @brief:节点安装服务
	@param [in] s ...:需要安装的所有服务

*/
func (node *Node) Setup(s ...service.IService) {
	for _, sv := range s {
		sv.OnSetup(sv)
		node.preSetupService = append(node.preSetupService, sv)
	}
}

/*
    @brief:开启服务监测器
	@param [in] interval:监测器每次报告的间隔
*/
func (node *Node) OpenProfilerReport(interval time.Duration) {
	node.profilerInterval = interval
}
