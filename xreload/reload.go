package xreload
import (
    "time"
    "os"
    "syscall"
    "os/signal"
    "fmt"
    "errors"
    "os/exec"
    "reflect"
    "log"
    "sync"
)

var reloadable *Reloader = nil

const (
    RELOAD_FLAG = "x_reload_flag"
)


var WaitTimeoutError = errors.New("timeout")

type Reloader struct {
    listener *ListenerReloadable
    reloadList []Reloadable
    Status int
    Stopped chan bool
    StopTimeout chan bool
    RestartDuration time.Duration
    StopDuration time.Duration
}


func NewReloadable(restartDuration, stopDuration time.Duration) *Reloader {
    if reloadable == nil {
        reloadable = &Reloader{}
        reloadable.reloadList = []Reloadable{}
        reloadable.Stopped = make(chan bool)
        reloadable.StopTimeout = make(chan bool)
        if restartDuration == 0 || stopDuration==0{
            return nil
        }
        reloadable.RestartDuration = restartDuration
        reloadable.StopDuration = stopDuration
        go reloadable.waitSignal()
    }
    return reloadable
}

func IsReloading() bool{
    flag := os.Getenv(RELOAD_FLAG)
    return flag!=""
}


func (this *Reloader)AddReloadble(item Reloadable){
    this.reloadList = append(this.reloadList, item)
}

func (this *Reloader)SetListener(l *ListenerReloadable) {
    this.listener = l
}

// Listener waits signal to kill or interrupt then restart.
func (this *Reloader)waitSignal() error {
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, syscall.SIGHUP, syscall.SIGTERM)
    for {
        sig := <-ch
        switch sig {
            case syscall.SIGTERM:
                return this.Stop()
            case  syscall.SIGHUP:
                return this.Restart()
        }
    }
    return nil // It'll never get here.
}

func (this *Reloader)Stop() error{
    this.Status = STATUS_STOPPING
    if this.listener != nil {
        this.listener.Stop()
    }
    for _, item := range this.reloadList{
        item.Stop()
    }
    //wait old process finish
    this.doWaitTimeout(this.StopDuration)
    return nil
}

func (this *Reloader)Restart() (err error){
    fmt.Println("restarting server")
    defer func(){
        log.Printf("restart finished. err:%v\n", err) 
    }()
   
    //do restart
    argv0, err := exec.LookPath(os.Args[0])
    if nil != err {
        return 
    }
    wd, err := os.Getwd()
    if nil != err {
        return 
    }
    v := reflect.ValueOf(this.listener.Listener).Elem().FieldByName("fd").Elem()
    fd := uintptr(v.FieldByName("sysfd").Int())
    // child fd order
    // 0:os.Stdin 1:os.Stdout 2:os.Stderr 3:listen socket
    allFiles := append([]*os.File{os.Stdin, os.Stdout, os.Stderr},
    os.NewFile(fd, "listen socket"))

    p, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
        Dir:   wd,
        Env:   append(os.Environ(), fmt.Sprintf("%s=%d", RELOAD_FLAG, fd)),
        Files: allFiles,
    })
    
    fmt.Printf("Start new process.pid[%d]\n", p.Pid)

    //stop current process
    this.Status = STATUS_RESTARTING
    if this.listener != nil {
        this.listener.Restart()
    }
    for _, item := range this.reloadList{
        item.Restart()
    }

    //wait old process finish
    this.doWaitTimeout(this.RestartDuration)
    return err
}

func (this *Reloader)AddReloadItem(item Reloadable) {
    this.reloadList = append(this.reloadList, item)
}

func (this *Reloader) WaitFinish() {
    ppid := os.Getpid()
  
    select {
    case <-this.Stopped:
        fmt.Printf("Process stopped. pid [%v]\n", ppid)
    case <-this.StopTimeout:
        fmt.Printf("Process stop timeout after:%v. pid [%v]\n", this.RestartDuration, ppid)
    }
    if this.listener != nil {
        s := this.listener.Status()
        this.printStatus(this.listener.GetId(), s)
    }
    for _, item := range this.reloadList{
        s := item.Status()
        this.printStatus(item.GetId(), s)
    }
}

func (this *Reloader) doWaitTimeout(t time.Duration){
    wg := sync.WaitGroup{}

    if this.listener != nil {
        go func(){
            wg.Add(1)
            this.listener.WaitTimeOut(t)
            wg.Done()
        }()
    }

    for _, item := range this.reloadList{
        go func(item Reloadable){
            wg.Add(1)
            item.WaitTimeOut(t)
            wg.Done()
        }(item)

    }

    allFinished := make(chan bool)
    go func() {
        wg.Wait()
        allFinished<-true
    }()

    select {
    case <-time.After(t):
        this.StopTimeout<-true
    case <-allFinished:
        this.Stopped<-true
    }
}

func (this *Reloader)printStatus(id string, s *ReloadStatus) {
    fmt.Printf("Id[%s] ReloadStatus:%s Err:%v Msg:%s\n", id, GetStatusStr(s.Status), s.Error, s.Msg);
}

// Kill current running os process.
func (this *Reloader) KillProcess() error {
    ppid := os.Getpid()
    if ppid == 1 { // init provided sockets, for example systemd
        return nil
    }
    p, err := os.FindProcess(ppid)
    if err != nil {
        return err
    }
    return p.Kill()
}