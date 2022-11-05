package mempool

import (
	"sync"
)
type IPoolData interface {
	Reset()
	IsRef()bool
	Ref()
	UnRef()
}
//通用内存池
type Pool struct {
	C chan interface{}  //缓存
	syncPool sync.Pool
}
//特化内存池
type PoolEx struct{
	C chan IPoolData  //缓存
	syncPool sync.Pool
}
/*
   @brief:生成一个通用内存池
   @param [in] c:缓存区
   @param [in] new:get时内存池为空时，调用的函数
   @return:内存池
*/
func NewPool(C chan interface{},New func()interface{}) *Pool{
	var p Pool
	p.C = C
	p.syncPool.New = New
	return &p
}
/*
    @brief:从内存池中取一个数据
*/
func (pool *Pool) Get() interface{}{
	select {
	case d := <-pool.C:
		return d
	default:
		//pool.c中为空，就从syncPool中重新生成一个
		return pool.syncPool.Get()
	}
}
/*
    @brief:回收一个数据
*/
func (pool *Pool) Put(data interface{}){
	select {
	case pool.C <- data:
	default:
		pool.syncPool.Put(data)
	}
}
/*
   @brief:生成一个特化内存池（保存实现了IPoolData的数据）
   @param [in] c:缓存区
   @param [in] new:get时内存池为空时，调用的函数
   @return:内存池
*/
func NewPoolEx(C chan IPoolData,New func()IPoolData) *PoolEx{
	var pool PoolEx
	pool.C = C
	pool.syncPool.New = func() interface{} {
		return New()
	}
	return &pool
}
/*
    @brief:从内存池中取一个数据
*/
func (pool *PoolEx) Get() IPoolData{
	select {
	case d := <-pool.C:
		if d.IsRef() {
			panic("Pool data is in use.")
		}
		d.Ref()
		return d
	default:
		//pool.c中为空，就从syncPool中重新生成一个
		data := pool.syncPool.Get().(IPoolData)
		if data.IsRef() {
			panic("Pool data is in use.")
		}
		data.Ref()
		return data
	}

}
/*
    @brief:回收一个数据
*/
func (pool *PoolEx) Put(data IPoolData){
	if data.IsRef() == false {
		panic("Repeatedly freeing memory")
	}
	//提前解引用，防止递归释放
	data.UnRef()
	data.Reset()
	//再次解引用，防止Rest时错误标记
	data.UnRef()
	select {
	case pool.C <- data:
	default:
		pool.syncPool.Put(data)
	}
}

