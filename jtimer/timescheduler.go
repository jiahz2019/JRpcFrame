package jtimer

import (
	"JRpcFrame/jlog"
	"math"
	"sync"
	"time"
)

const (
	MaxChanBuff  = 2048 //MaxChanBuff 默认缓冲触发函数队列大小
	MaxTimeDelay = 100  //MaxTimeDelay 默认最大误差时间
)

//全局定时器
var GlobelTimer *TimerScheduler

func init() {
	GlobelTimer = NewAutoTimerScheduler()
	jlog.StdLogger.Info("GlobelTimer start runing ")
}

//TimerScheduler 计时器调度器
type TimerScheduler struct {
	tw           *TimeWheel      //当前调度器的最高级时间轮
	IdGen        uint32          //定时器编号累加器
	triggerChan  chan *DelayFunc //已经触发定时器的channel
	sync.RWMutex                 //互斥锁
}

/*
   @brief:TimerScheduler的构造函数
*/
func NewTimerScheduler() *TimerScheduler {
	secondTw := NewTimeWheel(SecondName, SecondInterval, SecondScales, TimerMaxCap)
	minuteTw := NewTimeWheel(MinuteName, MinuteInterval, MinuteScales, TimerMaxCap)
	hourTw := NewTimeWheel(HourName, HourInterval, HourScales, TimerMaxCap)

	//将分层时间轮做关联
	hourTw.AddTimeWheel(minuteTw)
	minuteTw.AddTimeWheel(secondTw)

	//时间轮运行
	secondTw.Run()
	minuteTw.Run()
	hourTw.Run()

	return &TimerScheduler{
		tw:          hourTw,
		triggerChan: make(chan *DelayFunc, MaxChanBuff),
	}

}

/*
    @brief:在TimerScheduler中加入一个定点触发的timer
	@param [in] df:定时器延迟函数
	@param [in] unixNano:定时器触发的时间点
	@return : timer的id，error
*/
func (ts *TimerScheduler) CreateTimerAt(df *DelayFunc, unixNano int64) (uint32, error) {
	ts.Lock()
	defer ts.Unlock()

	ts.IdGen++
	return ts.IdGen, ts.tw.AddTimer(ts.IdGen, NewTimerAt(df, unixNano))

}

/*
    @brief:在TimerScheduler中加入一个延迟触发的timer
	@param [in] df:定时器延迟函数
	@param [in] duration:延迟时间段
	@param [in]  times:定时器使用次数
	@param [in]  interval:多次使用定时器时，每次的间隔
	@return : timer的id，error
*/
func (ts *TimerScheduler) CreateTimerAfter(df *DelayFunc, duration time.Duration, times int, interval int64) (uint32, error) {
	ts.Lock()
	defer ts.Unlock()

	ts.IdGen++
	return ts.IdGen, ts.tw.AddTimer(ts.IdGen, NewTimerAfter(df, duration, times, interval))

}

/*
    @brief:移除timer
	@param [in] tId :需要移除的timer id
*/
func (ts *TimerScheduler) RomoveTimer(tID uint32) {
	ts.Lock()
	ts.Unlock()
	tw := ts.tw
	for tw != nil {
		tw.RemoveTimer(tID)
		tw = tw.nextTimeWheel
	}
}

/*
   @brief:获取计时结束的延迟执行函数通道
*/
func (ts *TimerScheduler) GetTriggerChan() chan *DelayFunc {
	return ts.triggerChan
}

/*
   @brief:非阻塞的方式启动timerSchedule
*/
func (ts *TimerScheduler) Start() {
	go func() {
		for {
			now := UnixMilli()

			//获取最近误差MaxTimeDelay 毫秒的超时定时器集合
			timerList := ts.tw.GetTimerWithIn(MaxTimeDelay * time.Millisecond)

			for _, timer := range timerList {
				if (math.Abs(float64(now - timer.unixts))) > MaxTimeDelay {
					//定时器未在规定的时间内触发
					jlog.StdLogger.Error("want call at ", timer.unixts, "; real call at", now, "; delay ", now-timer.unixts)
				}
				ts.triggerChan <- timer.delayFunc
			}

			time.Sleep(MaxTimeDelay / 2 * time.Millisecond)
		}
	}()
}

/*
    @brief:生成一个自动调度时间轮调度器
	@return:创建的时间轮调度器
*/
func NewAutoTimerScheduler() *TimerScheduler {
	//创建并启动一个时间轮调度器
	autoScheduler := NewTimerScheduler()
	autoScheduler.Start()

	//从调度器中获取超时 触发的函数 并执行
	go func() {
		delayFuncChan := autoScheduler.GetTriggerChan()
		for df := range delayFuncChan {

			go df.Call()
		}
	}()
	return autoScheduler
}
