package mempool

import (

	"sync"
)

//切片内存池
type SlicePool struct {
	minAreaValue int //切片最小范围值
	maxAreaValue int //切片最大范围值
	growthValue  int //内存增长值
	//每个pool保存一种长度的[]byte，[]byte大小从minAreaValue依次递增growthValue，直到maxAreaValue
	pool         []sync.Pool
}
//slicepool集合，提供更多长度的[]byte
type SlicePoolList struct{
	poolNum int
	PoolList []*SlicePool
}
/*
    @brief:构造函数
    @param [in] poolnum:SlicePool的数量
	@param [in] args :SlicePool

*/
func NewSlicePoolList(poolNum int, args ...*SlicePool)*SlicePoolList{
	s:=&SlicePoolList{
		poolNum: poolNum,
		PoolList: make([]*SlicePool, poolNum),
	}
	for i := 0; i < len(args); i++ {
		s.PoolList[i]=args[i]
	}
	return s
}
/*
    @brief:构造函数
    @param [in] minAreaValue:切片最小范围值
	@param [in] maxAreaValue :切片最大范围值
	@param [in] growthValue  :内存增长值
*/
func NewSlicePool(minAreaValue int,maxAreaValue int,growthValue  int) *SlicePool {
	areaPool :=&SlicePool{
		minAreaValue: minAreaValue,
		maxAreaValue: maxAreaValue,
		growthValue: growthValue,
	}
	//初始化[]pool
	poolLen := (areaPool.maxAreaValue - areaPool.minAreaValue + 1) / areaPool.growthValue
	areaPool.pool = make([]sync.Pool, poolLen)
	for i := 0; i < poolLen; i++ {
		//初始化每个pool，pool存放一种[]byte
		memSize := (areaPool.minAreaValue - 1) + (i+1)*areaPool.growthValue
		areaPool.pool[i] = sync.Pool{New: func() interface{} {
			//fmt.Println("make memsize:",memSize)
			return make([]byte, memSize)
		}}
	}
	return areaPool
}
/*
    @brief:从池列表中取出一个大小为size的[]byte切片
	@param [in] size:取出[]byte切片的大小
*/
func (s *SlicePoolList)MakeByteSlice(size int) []byte {
	for i := 0; i < s.poolNum; i++ {
		//从三个不同大小的slicepool中选择
		if size <= s.PoolList[i].maxAreaValue {
			
			return s.PoolList[i].makeByteSlice(size)
		}
	}

	return make([]byte, size)
}
/*
    @brief:释放[]byte切片
	@param [in] byteBuff:需要释放的[]byte切片
	@return:释放是否成功
*/
func  (s *SlicePoolList)ReleaseByteSlice(byteBuff []byte) bool {
	for i := 0; i < s.poolNum; i++ {
		if cap(byteBuff) <= s.PoolList[i].maxAreaValue {

			return s.PoolList[i].releaseByteSlice(byteBuff)
		}
	}

	return false
}


func (areaPool *SlicePool) makeByteSlice(size int) []byte {
	//从slicepool中找到适合size大小[]byte的位置
	pos := areaPool.getPosByteSize(size)
	if pos > len(areaPool.pool) || pos == -1 {
		return nil
	}

	return areaPool.pool[pos].Get().([]byte)[:size]
}

func (areaPool *SlicePool) getPosByteSize(size int) int {
	pos := (size - areaPool.minAreaValue) / areaPool.growthValue
	if pos >= len(areaPool.pool) {
		return -1
	}

	return pos
}

func (areaPool *SlicePool) releaseByteSlice(byteBuff []byte) bool {
	pos := areaPool.getPosByteSize(cap(byteBuff))
	if pos > len(areaPool.pool) || pos == -1 {
		panic("assert!")
	}
	areaPool.pool[pos].Put(byteBuff)
	return true
}

