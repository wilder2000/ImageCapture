package pool

import (
	"errors"
	"time"
)

var (
	AppendTimeout time.Duration
)

// DocEngine 文件处理引擎
type DocEngine[T interface{}] struct {
	DocList chan *T       //文件例表
	timeout time.Duration //超时时间
}

// Append 添加一个文档到队列中，如果队列满了，会有一个超时时间，过了超时时间返回失败的错误
func (de *DocEngine[T]) Append(dInfo *T) (bool, error) {
	timeout := time.After(time.Second * de.timeout)
	for {
		select {
		case de.DocList <- dInfo:
			return true, nil
		case <-timeout:
			return false, errors.New("try to append doc to processed timeout")
		}
	}

}

// New 初始化创建一个文件处理引擎
func New[T interface{}](pooSize int, currTh int, proc ExeProcess[T]) *DocEngine[T] {
	AppendTimeout = 10
	dEngine := &DocEngine[T]{}
	dEngine.DocList = make(chan *T, pooSize)
	dEngine.timeout = AppendTimeout
	for i := 0; i < currTh; i++ {
		go run[T](proc, dEngine)
	}
	return dEngine
}

// 启动引警
func run[T interface{}](proc ExeProcess[T], engine *DocEngine[T]) {
	for {
		select {
		case doc := <-engine.DocList:
			proc(doc)

		case <-time.After(2 * time.Second):
			log.Logger.DebugF("wait 2 second. array length:%d", len(engine.DocList))
		}

	}

}

type ExeProcess[T interface{}] func(doc *T)
