package main

import (
    "net/http"
    "time"
    "fmt"
//    "net"
    "os"
    "xgracerestart/xreload"
    "xgracerestart/task"
    "sync/atomic"
)

type myHandler struct {

}
var taskId int32

func (this *myHandler)ServeHTTP(res http.ResponseWriter, req *http.Request){
    <-time.After(time.Second)
    e := task.AddTask(&task.Task{Id:taskId})
    
    if e!=nil{
        res.Write([]byte(e.Error()))        
    }else{
        atomic.AddInt32(&taskId, int32(1))
        res.Write([]byte("res ok."))
    }
}

func  main(){
    handler := &myHandler{}

    s := &http.Server{
        Handler:      handler,
        ReadTimeout:  time.Duration(1) * time.Second*2,
        WriteTimeout: time.Duration(1) * time.Second*2,
    }

//    addr := fmt.Sprintf("%s:%d", "127.0.0.1", 8888)
//    laddr, err := net.ResolveTCPAddr("tcp", addr)
//    if nil != err {
//        return
//    }
//
//    l, err := xreload.GetInitListener(laddr)
//    if err != nil {
//        return
//    }
//    listenerReloader := xreload.NewListenerReloader(l)

    listenerReloader, err := xreload.NewListenerWithAddr("tcp", "127.0.0.1:8888")
    if err != nil {
        fmt.Println("Listen error", err)
        os.Exit(-1)
    }
    
    reloader := xreload.NewReloadable(time.Second*10, time.Second*60)
    reloader.SetListener(listenerReloader)
    tsk := task.NewTaskReloadable()
    reloader.AddReloadble(tsk)
    
    fmt.Println("=======Server Start OK=======")
    
    go func(){
        s.Serve(listenerReloader)
    }()
    
    reloader.WaitFinish()
}