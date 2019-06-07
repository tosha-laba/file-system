package main

import "container/list"

// BlockManager представляет менеджер свободных дисковых секторов
type BlockManager struct {
	blockList *list.List
}

// NewBlockManager создает новый менеджер свободных блоков
func NewBlockManager() *BlockManager {
	blockList := list.New()
	for i := 0; i < MaxBlockCount; i++ {
		blockList.PushBack(int32(i))
	}
	return &BlockManager{blockList}
}

// GetFreeBlock пытается получить доступ к свободному блоку,
// если получается: возвращает адрес, иначе: -1
func (bm *BlockManager) GetFreeBlock() int32 {
	e := bm.blockList.Front()
	if e == nil {
		return -1
	}

	value := e.Value.(int32)
	bm.blockList.Remove(e)

	return value
}

// AddBlock помечает указанный блок свободным
func (bm *BlockManager) AddBlock(blockAddress int) {
	bm.blockList.PushFront(blockAddress)
}
