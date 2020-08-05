package main

import (
    "context"
    "fmt"
	"time"
)


type futureInterface interface {
    Cancel()
    Cancelled() bool
    Done() bool
    result() (interface{}, error)
    resultUntil(d time.Duration) (interface{}, bool, error)
    doneCallBack(func(interface{}) (interface{}, error)) futureInterface
}

// New creates a new Future that wraps the provided function.
func New(inFunc func() (interface{}, error)) futureInterface {
	return NewWithContext(context.Background(), inFunc)
}

func newInner(cancelChan <-chan struct{}, cancelFunc context.CancelFunc, inFunc func() (interface{}, error)) futureInterface {
	f := futureStruct{
		done:       make(chan struct{}),
		cancelChan: cancelChan,
		cancelFunc: cancelFunc,
	}
	go func() {
		go func() {
			f.val, f.err = inFunc()
			close(f.done)
		}()
		select {
		case <-f.done:
		case <-f.cancelChan:
		}
	}()
	return &f
}


func NewWithContext(ctx context.Context, inFunc func() (interface{}, error)) futureInterface {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	return newInner(cancelCtx.Done(), cancelFunc, inFunc)
}

type futureStruct struct {
	done       chan struct{}
	cancelChan <-chan struct{}
	cancelFunc context.CancelFunc
	val        interface{}
	err        error
}

func (f *futureStruct) Cancel() {
	select {
	case <-f.done:
		return
	case <-f.cancelChan:
		return
	default:
		f.cancelFunc()
	}
}

func (f *futureStruct) Cancelled() bool {
	select {
	case <-f.cancelChan:
		return true
	default:
		return false
	}
}

func (f *futureStruct) Done() bool {
    select {
    case <- f.done:
        return true
    case <- f.cancelChan:
        return true
    default:
        return false
    }
    return false
}

func (f *futureStruct) result() (interface{}, error) {
	select {
	case <-f.done:
		return f.val, f.err
	case <-f.cancelChan:
	}
	return nil, nil
}

func (f *futureStruct) resultUntil(d time.Duration) (interface{}, bool, error) {
	select {
	case <-f.done:
		val, err := f.result()
		return val, false, err
	case <-time.After(d):
		return nil, true, nil
	case <-f.cancelChan:
	}
	return nil, false, nil
}

func (f *futureStruct) doneCallBack(next func(interface{}) (interface{}, error)) futureInterface {
	nextFuture := newInner(f.cancelChan, f.cancelFunc, func() (interface{}, error) {
		result, err := f.result()
		if f.Cancelled() || err != nil {
			return result, err
		}
		return next(result)
	})
	return nextFuture
}


// Main func to test above functions.
func main(){
    var tempVal = 200
    longTimeFunc := func(tempVal int) (int, error) {
   		time.Sleep(5 * time.Second)
   		return tempVal * 2, nil
   	}

    //starts new instance of this future implementation
   	f := New(func() (interface{}, error) {
   		return longTimeFunc(tempVal)
   	})

    // Checking cancel call
   	go func() {
   		time.Sleep(2 * time.Second)
   		f.Cancel()
   	}()
    result, err := f.result()
   	fmt.Println(result, err, f.Cancelled())

    // Checking get call
    g := New(func() (interface{}, error) {
       		return longTimeFunc(tempVal)
       	})
    gResult, gErr := g.result()

   	fmt.Println(g.Done(), gResult, gErr, g.Cancelled())
}