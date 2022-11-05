package profiler

import(
	"container/list"
	"sync"
	"time"
)

//最大超长时间，一般可以认为是死锁或者死循环，或者极差的性能问题
var DefaultMaxOvertime time.Duration = 5*time.Second
//超过该时间将会监控报告
var DefaultOvertime time.Duration = 10*time.Millisecond
var DefaultMaxRecordNum int = 100 //最大记录条数
//报告类型
type RecordType int
const  (
	MaxOvertimeType = 1
	OvertimeType    =2
	)

//
type Element struct {
	tagName string
	pushTime time.Time
}
//报告信息
type Record struct {
	RType RecordType
	CostTime time.Duration
	RecordName string
}
//监测器
type Profiler struct {
	stack *list.List           //保存需要监测的元素
	stackLocker sync.RWMutex
	record *list.List          //保存报告信息

	maxOverTime time.Duration  //最大超长时间
	overTime time.Duration     //超过该时间将会监控报告
	maxRecordNum int          //最大记录条数

	
	callNum int                //调用次数
	totalCostTime time.Duration //总消费时间长
	
}
//分析器
type Analyzer struct {
	elem *list.Element
	profiler *Profiler
}

func init(){
	mapProfiler = map[string]*Profiler{}
}

/*
    @brief:监测器构造函数
*/
func NewProfiler()*Profiler{
	p:= &Profiler{
		stack:list.New(),
		record:list.New(),
		maxOverTime: DefaultMaxOvertime,
		overTime: DefaultOvertime,
	}
	return p
}


/*
    @brief:压入需要监测的信息
	@param [in] tag:需要监测的信息标签
	@return: 需要监测的信息对应的分析器
*/
func (slf *Profiler) Push(tag string) *Analyzer{
	slf.stackLocker.Lock()
	defer slf.stackLocker.Unlock()

	pElem := slf.stack.PushBack(&Element{tagName:tag,pushTime:time.Now()})

	return &Analyzer{elem:pElem,profiler:slf}
}
/*
    @brief:分析器弹出已经监测完毕的信息，将其作为record的形式保存在profiler中
*/
func (slf *Analyzer) Pop(){
	slf.profiler.stackLocker.Lock()
	defer slf.profiler.stackLocker.Unlock()

	pElement := slf.elem.Value.(*Element)
	//检查监测信息
	pElem,subTm := slf.profiler.check(pElement)
	slf.profiler.callNum+=1
	slf.profiler.totalCostTime += subTm
	if pElem != nil {
		//保存
		slf.profiler.pushRecordLog(pElem)
	}
	slf.profiler.stack.Remove(slf.elem)
}
/*
    @brief:检查监测信息是否进行记录
	@param [in] pElem:需要检查的监测信息
	@return:监测记录，监测持续时间
*/
func (slf *Profiler) check(pElem *Element) (*Record,time.Duration) {
	if pElem == nil {
		return nil,0
	}

	subTm := time.Now().Sub(pElem.pushTime)
	//若运行时间小于每次监控报告的时间，则不报告该监测信息
	if subTm < slf.overTime {
		return nil,subTm
	}

	record := Record{
		RType:      OvertimeType,
		CostTime:   subTm,
		RecordName: pElem.tagName,
	}

	if subTm>slf.maxOverTime {
		record.RType = MaxOvertimeType
	}

	return &record,subTm
}
/*
    @brief:压入监测记录
	@param [in] record:需要压入的监测记录
*/
func (slf *Profiler) pushRecordLog(record *Record){
	if slf.record.Len()>= DefaultMaxRecordNum {
		front := slf.stack.Front()
		if front!=nil {
			slf.stack.Remove(front)
		}
	}

	slf.record.PushBack(record)
}


func (slf *Profiler) SetMaxOverTime(tm time.Duration){
	slf.maxOverTime = tm
}

func (slf *Profiler) SetOverTime(tm time.Duration){
	slf.overTime = tm
}

func (slf *Profiler) SetMaxRecordNum(num int){
	slf.maxRecordNum = num
}