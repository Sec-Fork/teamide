package task

import (
	"go.uber.org/zap"
	"sync"
	"teamide/pkg/util"
	"testing"
	"time"
)

func TestAddTask(t *testing.T) {
	util.Logger.Info("TestAddTask Start", zap.Any("Time", time.Now()))
	wg := &sync.WaitGroup{}
	wg.Add(1)

	task := &CronTask{}

	task.Key = "xxx"
	task.Spec = "*/5 * * * * ?"
	task.ExecutionTimes = 2
	task.Do = func() {
		util.Logger.Info("执行", zap.Any("Time", time.Now()))
	}
	task.DoEnd = func() {
		util.Logger.Info("执行结束", zap.Any("Time", time.Now()))
		wg.Done()
	}

	err := AddTask(task)

	if err != nil {
		panic(err)
	}
	select {
	case <-time.After(1 * time.Minute):
		break
	case <-wait(wg):
		break
	}
	util.Logger.Info("TestAddTask End", zap.Any("Time", time.Now()))
}

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}
