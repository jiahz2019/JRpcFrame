package profiler

import (
	"JRpcFrame/jlog"
	"container/list"
	"fmt"
	"time"
)

var mapProfiler map[string]*Profiler

func init() {
	mapProfiler = map[string]*Profiler{}
}

/*
    @brief:注册一个profiler
	@param [in] profilerName :监测器名字
*/
func RegProfiler(profilerName string) *Profiler {
	if _, ok := mapProfiler[profilerName]; ok == true {
		return nil
	}

	pProfiler := NewProfiler()
	mapProfiler[profilerName] = pProfiler
	return pProfiler
}

func Report() {
	var record *list.List
	for name, prof := range mapProfiler {
		prof.stackLocker.RLock()

		//取栈顶，是否存在异常MaxOverTime数据
		pElem := prof.stack.Back()
		for pElem != nil {
			pElement := pElem.Value.(*Element)
			pExceptionElem, _ := prof.check(pElement)
			if pExceptionElem != nil {
				prof.pushRecordLog(pExceptionElem)
			}
			pElem = pElem.Prev()
		}

		if prof.record.Len() == 0 {
			prof.stackLocker.RUnlock()
			continue
		}

		record = prof.record
		prof.record = list.New()
		prof.stackLocker.RUnlock()

		DefaultReportFunction(name, prof.callNum, prof.totalCostTime, record)
	}
}

func DefaultReportFunction(name string, callNum int, costTime time.Duration, record *list.List) {
	if record.Len() <= 0 {
		return
	}

	var strReport string
	strReport = "Profiler report tag " + name + ":\n"
	var average int64
	if callNum > 0 {
		average = costTime.Milliseconds() / int64(callNum)
	}

	strReport += fmt.Sprintf("process count %d,take time %d Milliseconds,average %d Milliseconds/per.\n", callNum, costTime.Milliseconds(), average)
	elem := record.Front()
	var strTypes string
	for elem != nil {
		pRecord := elem.Value.(*Record)
		if pRecord.RType == MaxOvertimeType {
			strTypes = "too slow process"
		} else {
			strTypes = "slow process"
		}

		strReport += fmt.Sprintf("%s:%s is take %d Milliseconds\n", strTypes, pRecord.RecordName, pRecord.CostTime.Milliseconds())
		elem = elem.Next()
	}

	jlog.StdLogger.Info(strReport)
}
