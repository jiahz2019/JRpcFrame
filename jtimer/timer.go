package jtimer

import (
	"time"
)

//Timer 定时器实现
type Timer struct {
	delayFunc *DelayFunc	//延迟调用函数
	unixts int64            //调用时间(unix 时间， 单位ms)

	Interval  int64         //多次使用定时器时，每次的间隔
	times  int             //定时器使用次数
}

/*
    @brief:返回1970-1-1至今经历的毫秒数
*/
func UnixMilli() int64 {
	return time.Now().UnixNano() / 1e6
}

/*
    @brief:创建一个定时器,在指定的时间触
	@param [in]  df:DelayFunc类型的延迟调用函数类型
	@param [in]  unixNano: unix计算机从1970-1-1至今经历的纳秒数
*/
func NewTimerAt(df *DelayFunc, unixNano int64) *Timer {
	return &Timer{
		delayFunc: df,
		unixts:    unixNano / 1e6, //将纳秒转换成对应的毫秒 ms ，定时器以ms为最小精度
        Interval: 0,
		times:1,
	}
}

/*
    @brief:创建一个定时器，在当前时间延迟duration之后触发
	@param [in]  df:DelayFunc类型的延迟调用函数类型
	@param [in]  duration: 延迟时间
	@param [in]  times:定时器使用次数
	@param [in]  interval:多次使用定时器时，每次的间隔

*/
func NewTimerAfter(df *DelayFunc, duration time.Duration,times int,interval int64) *Timer {
	t:=NewTimerAt(df, time.Now().UnixNano()+int64(duration))
	t.SetTimes(times)
	t.SetInterval(interval)
	return t
}

/*
    @brief:设置定时器循环调用的次数
*/
func (t *Timer)SetTimes(times int){
    t.times=times
}


/*
    @brief:设置定时器每次的间隔
*/
func (t *Timer)SetInterval(interval int64){
    t.Interval=interval
}

/*
    @brief:启动定时器，用一个go承载
*/
func (t *Timer) Run() {
	go func() {
		now := UnixMilli()
		if t.unixts > now {
			//睡眠，直至时间超时,已微秒为单位进行睡眠
			time.Sleep(time.Duration(t.unixts-now) * time.Millisecond)
		}
		//调用事先注册好的超时延迟方法
		t.delayFunc.Call()
	}()
}