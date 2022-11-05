package jtimer

import (
	"JRpcFrame/jlog"
	"errors"
	"fmt"
	"sync"
	"time"
)

//默认3个时间轮参数,以毫秒为单位
const (
	HourName     = "HOUR"
	HourInterval = 60 * 60 * 1e3
	HourScales   = 12

	MinuteName     = "MINUTE"
	MinuteInterval = 60 * 1e3
	MinuteScales   = 60

	SecondName     = "SECOND"
	SecondInterval = 1e3
	SecondScales   = 60

	//每个时间轮刻度挂载定时器的最大个数
	TimerMaxCap = 2048
)

type TimeWheel struct {
	name     string //时间轮名字
	interval int64  //刻度间隔
	scales   int    //每个时间轮上的刻度数
	curIndex int    //当前时间指针的指向
	maxCap   int    //当前时间轮上的所有timer

	//int是当前时间轮的刻度,map[uint32]表示定时器链表(timerid---timer)
	timerQueue    map[int]map[uint32]*Timer
	nextTimeWheel *TimeWheel //下一层时间轮
	sync.RWMutex
}

/*
   @brief:timewheel构造函数
   @param [in] name: 时间轮名字
   @param [in] interval: 每个刻度大小
   @param [in] scales: 总共刻度数
   @param [in] maxcap: 每个刻度的所能有的最大定时器个数
*/
func NewTimeWheel(name string, interval int64, scales int, maxCap int) *TimeWheel {
	tw := &TimeWheel{
		name:       name,
		interval:   interval,
		scales:     scales,
		maxCap:     maxCap,
		timerQueue: make(map[int]map[uint32]*Timer, scales),
	}
	//初始化timerQueue
	for i := 0; i < scales; i++ {
		tw.timerQueue[i] = make(map[uint32]*Timer, maxCap)
	}
	return tw
}

/*
    @brief:添加定时器到一个分层时间轮中
	@param [in] tId:定时器id
    @param [in] t:定时器
	@param [in] isInRun:是否是在run中addtimer
*/
func (tw *TimeWheel) addTimer(tID uint32, t *Timer, forceNext bool) error {
	defer func() error {
		if err := recover(); err != nil {
			errstr := fmt.Sprintf("addTimer function err : %s", err)
			jlog.StdLogger.Info(errstr)
			return errors.New(errstr)
		}
		return nil
	}()

	//得到当前的超时时间间隔(ms)毫秒为单位
	delayInterval := t.unixts - UnixMilli()

	//如果当前的超时时间 大于一个刻度的时间间隔
	if delayInterval >= tw.interval {

		//在对应的刻度上的加入当前定时器
		dn := delayInterval / tw.interval
		tw.timerQueue[(tw.curIndex+int(dn))%tw.scales][tID] = t
		return nil

	}
	//如果当前的超时时间,小于一个刻度的时间间隔，并且当前时间轮没有下一层，
	//即刻度度最小的时间轮
	if delayInterval < tw.interval && tw.nextTimeWheel == nil {
		//强制移至下一个刻度
		//由于这是最小的时间轮，如果时间轮刻度已经过去，不强制把该定时器Timer移至下时刻，就永远不会被取走并触发调用
		if forceNext {
			tw.timerQueue[(tw.curIndex+1)%tw.scales][tID] = t
		} else {
			tw.timerQueue[tw.curIndex][tID] = t
		}
		return nil
	}
	//如果当前的超时时间，小于一个刻度的时间间隔，并且有下一层时间轮
	if delayInterval < tw.interval {
		return tw.nextTimeWheel.AddTimer(tID, t)
	}

	return nil
}

/*
   @brief:时间轮添加定时器
*/
func (tw *TimeWheel) AddTimer(tID uint32, t *Timer) error {
	defer tw.Unlock()
	tw.Lock()
	return tw.addTimer(tID, t, false)
}

/*
    @brief:删除一个定时器，根据定时器的ID
	@param [in] tId:定时器id
*/
func (tw *TimeWheel) RemoveTimer(tID uint32) {
	tw.Lock()
	defer tw.Unlock()

	for i := 0; i < tw.scales; i++ {
		if _, ok := tw.timerQueue[i][tID]; ok {
			delete(tw.timerQueue[i], tID)
		}
	}
}

/*
    @brief:给一个时间轮添加下层时间轮
	@param [in] next:需要添加的时间轮
*/
func (tw *TimeWheel) AddTimeWheel(next *TimeWheel) {
	tw.nextTimeWheel = next
}

/*
   @brief:时间轮运行
*/
func (tw *TimeWheel) run() {
	for {
		//时间轮每间隔interval一刻度时间，触发转动一次
		time.Sleep(time.Duration(tw.interval) * time.Millisecond)
		tw.Lock()

		//取出挂载在当前刻度的全部定时器,给当前刻度再重新开辟一个map Timer容器
		curTimers := tw.timerQueue[tw.curIndex]
		tw.timerQueue[tw.curIndex] = make(map[uint32]*Timer, tw.maxCap)
		for tID, timer := range curTimers {
			//时间轮走到该刻度，表示该刻度的所有定时器的该刻度时间都已走完，需将它们移至刻度更小的时间轮
			tw.addTimer(tID, timer, true)
		}

		//当前刻度指针走一格
		tw.curIndex = (tw.curIndex + 1) % tw.scales
		tw.Unlock()
	}

}

/*
   @brief:以异步协程运行时间轮
*/
func (tw *TimeWheel) Run() {
	go tw.run()
}

/**

    @brief:获取定时器在一段时间间隔内的Timer
	@return :满足条件的timer
**/
func (tw *TimeWheel) GetTimerWithIn(duration time.Duration) map[uint32]*Timer {
	//最终触发定时器的一定是挂载最底层时间轮上的定时器
	// 找到最底层时间轮
	leaftw := tw
	for leaftw.nextTimeWheel != nil {
		leaftw = leaftw.nextTimeWheel
	}

	leaftw.Lock()
	defer leaftw.Unlock()
	//返回的Timer集合
	timerList := make(map[uint32]*Timer)

	now := UnixMilli()
	//取出当前时间轮刻度内全部Timer
	for tID, timer := range leaftw.timerQueue[leaftw.curIndex] {
		if timer.unixts-now < int64(duration/1e6) {

			timerList[tID] = timer

			if timer.times <= 1 {
				delete(leaftw.timerQueue[leaftw.curIndex], tID)
			} else {
				//多次使用的定时器删除后再添加
				newtimer := NewTimerAfter(timer.delayFunc, time.Duration(timer.Interval), timer.times-1, timer.Interval)
				delete(leaftw.timerQueue[leaftw.curIndex], tID)
				//注意解锁，以免死锁
				leaftw.Unlock()
				tw.AddTimer(tID, newtimer)
				leaftw.Lock()

			}

		}
	}

	return timerList
}
