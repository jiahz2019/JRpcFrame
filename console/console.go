package console

import(
	"fmt"
	"flag"
	"os"
)
//控制台上执行的程序名
var programName string
//执行对应命令的回调函数
type CommandFunctionCB func(args interface{}) error
//所有命令集合
var commandList []*Command
//命令值类型
type commandValueType int
const(
	boolType commandValueType = iota
	stringType commandValueType= iota
)

type Command struct{
	valType commandValueType
	name string              //命令的名字
	bValue  bool             //命令的值
	strValue  string         //命令的值
	usage  string            //命令备注
	cmdCb CommandFunctionCB
}

/*
    @brief: 执行命令
*/
func (cmd *Command) execute() error{
	if cmd.valType == boolType {
		return cmd.cmdCb(cmd.bValue)
	}else if cmd.valType == stringType {
		return cmd.cmdCb(cmd.strValue)
	}else{
		return fmt.Errorf("Unknow command type.")
	}
}

/*
    @brief: 根据参数，执行相应的回调函数
	@param [in] args:控制台输入参数
*/
func Run(args []string) error {
	flag.Parse()
	programName = args[0]
	if flag.NFlag() <= 0 {
		return fmt.Errorf("Command input parameter error,try `%s -help` for help",args[0])
	}

	var startCmd *Command
	for _,val := range commandList {
		//开始命令最后执行
		if val.name == "start" {
			startCmd = val
			continue
		}
		err := val.execute()
		if err != nil {
			return err
		}
	}
    //执行开始命令
	if startCmd != nil {
		return startCmd.execute()
	}

	return fmt.Errorf("Command input parameter error,try `%s -help` for help",args[0])
}

/*
    @brief: 对控制台注册值为bool的命令
	@param [in] cmdName:命令名字
	@param [in] defaultValue:命令默认值
	@param [in] usage :命令备注
	@param [in] fn:命令回调函数
*/
func RegisterCommandBool(cmdName string, defaultValue bool, usage string,fn CommandFunctionCB){
	var cmd Command
	cmd.valType = boolType
	cmd.name = cmdName
	cmd.cmdCb = fn
	cmd.usage = usage
	flag.BoolVar(&cmd.bValue, cmdName, defaultValue, usage)
	commandList = append(commandList,&cmd)
}
/*
    @brief: 对控制台注册值为string的命令
	@param [in] cmdName:命令名字
	@param [in] defaultValue:命令默认值
	@param [in] usage :命令备注
	@param [in] fn:命令回调函数
*/
func RegisterCommandString(cmdName string, defaultValue string, usage string,fn CommandFunctionCB){
	var cmd Command
	cmd.valType = stringType
	cmd.name = cmdName
	cmd.cmdCb = fn
	cmd.usage = usage
	flag.StringVar(&cmd.strValue, cmdName, defaultValue, usage)
	commandList = append(commandList,&cmd)
}
/*
    @brief: 控制台打印的所有默认命令的备注
*/
func PrintDefaults(){
	fmt.Fprintf(os.Stderr, "Options:\n")

	for _,val := range commandList {
		fmt.Fprintf(os.Stderr, "  -%-10s%10s\n",val.name,val.usage)
	}
}