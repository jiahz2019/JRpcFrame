package jlog
import (
	"bytes"

	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)


const(

	LOG_MAX_BUF=1024*1024
)

//日志头部信息标记位，采用bitmap方式，用户可以选择头部需要哪些标记位被打印
const (
	Date         =iota                                  //日期标记位  2019/01/23
	Time                                                //时间标记位  01:23:12
	MicroSeconds                                        //微秒级标记位 01:23:12.111222
	LongFile                                            //完整文件名称 /home/go/src/zinx/server.go
	ShortFile                                           //最后文件名   server.go
	LogLevel                                            //当前日志级别： 0(Debug), 1(Info), 2(Warn), 3(Error), 4(Panic), 5(Fatal)
	StdFlag      = Date | Time                          //标准头部日志格式
	DefaultFlag      = LogLevel | ShortFile | StdFlag       //默认日志头部格式
)

//日志等级
const (
    LogDebug=0
	LogInfo=1
	LogWarn=2
	LogError=3
	LogPanic=4
	LogFatal=5
    	
)

//日志级别对应的显示字符串
var levels = []string{
	"[DEBUG]",
	"[INFO]",
	"[WARN]",
	"[ERROR]",
	"[PANIC]",
	"[FATAL]",
}


//日志类
type Logger struct{
	mu sync.Mutex            //确保多协程读写文件
    prefix string            //日志前缀，包括时间，日志等级等信息
	flag int                 //日志是否输出头部信息
	out io.Writer            //日志输出的文件描述符
	buf bytes.Buffer         //输出的缓冲区
	outFile *os.File            //当前日志绑定的输出文件
	debugClose bool          //是否打印调试debug信息
	calldDepth int
	level      int            //日志器等级

}

/*
    @brief: 创建一个日志类
    @param [in] out: 标志输出的文件io
    @param [in] prefix: 日志前缀
 	@param [in] flag: 日志前缀标记位
  	@param [in] level: 日志等级
 
**/
func NewLogger(out io.Writer,prefix string,flag int,level int) *Logger{
    //默认 debug打开， calledDepth深度为2,JLogger对象调用日志打印方法最多调用两层到达output函数
	
	hlog:=&Logger{out:out,prefix: prefix,flag: flag,outFile: nil,debugClose: false,calldDepth:2,level:level }
    return hlog
}

/*
   @brief:回收日志处理
*/
func CleanJLogger(log *Logger) {
	log.closeFile()
}

/*
    @brief: 日志头格式化，header="<"+prefix+">"+time+logLevel+filename
    @param [in] t: 日志输出时间
	@param [in] file: 日志运行的文件名
	@param [in] line: 日志运行的文件行数
	@param [in] level: 日志等级

*/
func (log *Logger) formatHeader(t time.Time, file string, line int, level int){
	var buf *bytes.Buffer = &log.buf
	//如果当前前缀字符串不为空，那么需要先写前缀
	if log.prefix != "" {
		buf.WriteByte('<')
		buf.WriteString(log.prefix)
		buf.WriteByte('>')
	}
	//时间相关标志位被设置
  	if log.flag&(Date|Time|MicroSeconds)!=0{
        //日期位被标识
        if log.flag&Date!=0{
			year,month,day:=t.Date()
			buf.WriteString(strconv.Itoa(year)+"/")
			buf.WriteString(strconv.Itoa(int(month))+"/")
			buf.WriteString(strconv.Itoa(day))
			buf.WriteByte(' ')
		}
		//时钟位被标记
		if log.flag&(Time|MicroSeconds) != 0 {
			hour, min, sec := t.Clock()
			buf.WriteString(strconv.Itoa(hour)+":")
			buf.WriteString(strconv.Itoa(min)+":")
			buf.WriteString(strconv.Itoa(sec))
			//微秒被标记
			if log.flag&MicroSeconds != 0 {
				buf.WriteByte('.')
				buf.WriteString(strconv.Itoa(t.Nanosecond()/1e3)) // "11:15:33.123123
			}
			buf.WriteByte(' ')
		}
    } 
	// 日志级别位被标记
	if log.flag&LogLevel != 0 {
		buf.WriteString(levels[level])
	}
	//日志当前代码调用文件名名称位被标记
	if log.flag&(ShortFile|LongFile) != 0 {
		//短文件名称
		if log.flag&ShortFile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					//找到最后一个'/'之后的文件名称  如:/home/go/src/zinx.go 得到 "zinx.go"
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		buf.WriteString(file)
		buf.WriteByte(':')
		buf.WriteString(strconv.Itoa(line))//行数
		buf.WriteString(": ")
	}
}

/*
    @brief: 日志(header+s)输出到log.out中
	@param [in] level: 日志等级
	@param [in] s: 日志具体内容
*/
func (log *Logger) OutPut(level int,s string) error {
	if level<log.level{
	    return nil
	}
	now := time.Now() // 得到当前时间
	var file string   //当前调用日志接口的文件名称
	var line int      //当前代码行数
	var ok bool

    //得到当前调用者的文件名称和执行到的代码行数
    _, file, line, ok = runtime.Caller(log.calldDepth)
	if !ok {
		file = "unknown-file"
		line = 0
	}
	log.mu.Lock()
	defer log.mu.Unlock()

	log.buf.Reset()
	log.formatHeader(now,file,line,level)
	log.buf.WriteString(s)
	//补充回车
	if len(s)>0 && s[len(s)-1]!='\n'{
		log.buf.WriteByte('\n')
	}

	//将填充好的buf 写到IO输出上
	
	if log.out!=nil{
		_, err := log.out.Write(log.buf.Bytes())
		if err!=nil{
			return err
		}
	}
	if log.outFile!=nil{
		_, err := log.outFile.Write(log.buf.Bytes())
		if err!=nil{
			return err
		}
	}
	return nil

}

// ====> Debug <====
func (log *Logger) Debugf(format string, v ...interface{}) {
	if log.debugClose  {
		return
	}
	_ = log.OutPut(LogDebug, fmt.Sprintf(format, v...))
}

func (log *Logger) Debug(v ...interface{}) {
	if log.debugClose  {
		return
	}
	_ = log.OutPut(LogDebug, fmt.Sprintln(v...))
}

// ====> Info <====
func (log *Logger) Infof(format string, v ...interface{}) {
	_ = log.OutPut(LogInfo, fmt.Sprintf(format, v...))
}

func (log *Logger) Info(v ...interface{}) {
	_ = log.OutPut(LogInfo, fmt.Sprintln(v...))
}

// ====> Warn <====
func (log *Logger) Warnf(format string, v ...interface{}) {
	_ = log.OutPut(LogWarn, fmt.Sprintf(format, v...))
}

func (log *Logger) Warn(v ...interface{}) {
	_ = log.OutPut(LogWarn, fmt.Sprintln(v...))
}

// ====> Error <====
func (log *Logger) Errorf(format string, v ...interface{}) {
	_ = log.OutPut(LogError, fmt.Sprintf(format, v...))
}

func (log *Logger) Error(v ...interface{}) {
	_ = log.OutPut(LogError, fmt.Sprintln(v...))
}

// ====> Fatal 需要终止程序 <====
func (log *Logger) Fatalf(format string, v ...interface{}) {
	_ = log.OutPut(LogFatal, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (log *Logger) Fatal(v ...interface{}) {
	_ = log.OutPut(LogFatal, fmt.Sprintln(v...))
	os.Exit(1)
}

// ====> Panic  <====
func (log *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = log.OutPut(LogPanic, s)
	panic(s)
}

func (log *Logger) Panic(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = log.OutPut(LogPanic, s)
	panic(s)
}


func (log *Logger) CloseDebug() {
	log.debugClose = true
}

func (log *Logger) OpenDebug() {
	log.debugClose = false
}

/*
    @brief: 将堆栈信息输出到log.bug中
	@param [in] v: 日志具体内容

*/
func (log *Logger) Stack(v ...interface{}) {
	s := fmt.Sprint(v...)
	s += "\n"
	buf := make([]byte, LOG_MAX_BUF)
	n := runtime.Stack(buf, true) //得到当前堆栈信息
	s += string(buf[:n])
	s += "\n"
	_ = log.OutPut(LogError, s)
}

/*
    @brief: 获取当前日志flag
	@return: log.flag

*/
func (log *Logger)Flags()int {
	log.mu.Lock()
	defer log.mu.Unlock()
	return log.flag

}

/*
    @brief: 重置当前日志flag
	@param [in] flag: flag
*/
func (log *Logger) ResetFlags(flag int) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.flag = flag
}

/*
    @brief: 添加日志flag
	@param [in] flag: flag
*/
func (log *Logger) AddFlag(flag int) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.flag |= flag
}

/*
    @brief: 设置日志等级
	@param [in] level: 日志等级
*/
func (log *Logger) SetLevel(level int) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.level = level
}

/*
    @brief: 设置日志前缀
	@param [in] prefix: prefix
*/
func (log *Logger) SetPrefix(prefix string) {
	log.mu.Lock()
	defer log.mu.Unlock()
	log.prefix = prefix
}

/*
    @brief: 设置日志输出文件
	@param [in] fileDir: 文件的路径
	@param [in] fileName: 文件名字 
*/
func (log *Logger) SetLogFile(fileDir string, fileName string)error{

    var file *os.File

	fullPath:=fileDir+"/"+fileName
    var err error
	if log.checkFileExist(fullPath){
		//文件存在，打开
		file, err = os.OpenFile(fullPath, os.O_APPEND|os.O_RDWR, 0644)
	}else {
		//文件不存在，创建
		file, err = os.OpenFile(fullPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	}
	if err!=nil{
		return err
	}
	log.mu.Lock()
	defer log.mu.Unlock()

	//关闭之前绑定的文件
	log.closeFile()
	log.outFile = file
	return nil

}

/*
    @brief: 关闭日志绑定的文件
*/
func (log *Logger) closeFile() {
	if log.outFile != nil {
		_ = log.outFile.Close()
		log.outFile = nil
		log.out = os.Stderr
	}
}

/*
    @brief: 设置输出流
	@param: 设置的输出流
*/
func (log *Logger) SetOutPut(out *io.Writer ){
	log.mu.Lock()
	defer log.mu.Unlock()
    log.out=*out
}



/*
    @brief:判断日志文件是否存在
	@param [in] filename:文件路径+文件名
	@return: 是否存在
*/
func (log *Logger) checkFileExist(filename string) bool {
	exist := true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

/*
    @brief:如果文件路径没有就创建
	@param [in] dir:路径
*/
func mkdirLog(dir string) (e error) {
	_, er := os.Stat(dir)
	b := er == nil || os.IsExist(er)
	if !b {
		if err := os.MkdirAll(dir, 0775); err != nil {
			if os.IsPermission(err) {
				e = err
			}
		}
	}
	return
}