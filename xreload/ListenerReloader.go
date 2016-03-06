package xreload
import (
    "time"
    "net"
    "sync"
    "os"
    "errors"
    "fmt"
)

// Allows for us to notice when the connection is closed.
type conn struct {
    net.Conn
    wg      *sync.WaitGroup
    Isclosed bool
    lock    sync.Mutex
}

// Close current processing connection.
func (c conn) Close() error {
    c.lock.Lock()
    defer c.lock.Unlock()
    err := c.Conn.Close()
    if !c.Isclosed && err == nil {
        c.wg.Done()
        c.Isclosed = true
    }
    return err
}


// Get current net.Listen in running process.
func GetInitListener(tcpaddr *net.TCPAddr) (l net.Listener, err error) {
    if !IsReloading() {
        return net.ListenTCP("tcp", tcpaddr)
    }

    // 0:os.Stdin 1:os.Stdout 2:os.Stderr 3:listen socket
    f := os.NewFile(uintptr(3), "listen socket")
    l, err = net.FileListener(f)
    if err != nil {
        return nil, err
    }
    return l, nil
}

type ListenerReloadable struct {
    net.Listener
    count   int64
    status  int
    wg      sync.WaitGroup
}

func NewListenerWithAddr(proc string, addr string) (ret *ListenerReloadable, err error){
    laddr, err := net.ResolveTCPAddr(proc, addr)
    if nil != err {
        return
    }
    l, err := GetInitListener(laddr)
    if err != nil {
        return 
    }
    r := &ListenerReloadable{Listener:l}
    
    return r, nil
}

func NewListenerReloader(l net.Listener) *ListenerReloadable {
    r := &ListenerReloadable{Listener:l}
    return r
}

func (this *ListenerReloadable)GetId() string{
    return "listener"
}

func (this *ListenerReloadable)Stop() error{
    this.status = STATUS_STOPPING
    return nil
}

func (this *ListenerReloadable)Restart() error{
    this.status = STATUS_RESTARTING
    return nil
}

//等待超时
func (this *ListenerReloadable) Wait() error{
    this.wg.Wait()
    this.status = STATUS_STOPPED
    return nil
}

//等待超时
func (this *ListenerReloadable) WaitTimeOut(t time.Duration) error{
    timeout := time.NewTimer(t)
    wait := make(chan struct{})
    go func() {
        this.wg.Wait()
        this.status = STATUS_STOPPED
        wait <- struct{}{}
    }()

    select {
    case <-timeout.C:
        return WaitTimeoutError
    case <-wait:
        return nil
    }
}

func (this *ListenerReloadable) Status() *ReloadStatus {
    info := &ReloadStatus{Error:nil, Status:this.status}
    switch this.status {
        case STATUS_STOPPED:
        info.Msg = fmt.Sprintf("Listner task stoped")
        case STATUS_STOP_TIMEOUT:
        info.Error = errors.New("Listner stop time out")
        default:
        info.Msg = fmt.Sprintf("Listner task running")
    }
    return info
}

//Listener interface
// Set stopped Listener to accept requests again.
// it returns the accepted and closable connection or error.
func (this *ListenerReloadable) Accept() (c net.Conn, err error) {
    fmt.Println("Accept new conn status:", this.status)
    if this.status != STATUS_RUNNING{
        return nil, errors.New(fmt.Sprintf("Listener is not running.status:%d", this.status))
    }
    c, err = this.Listener.Accept()
    if err != nil {
        return
    }
    this.wg.Add(1)
    // Wrap the returned connection, so that we can observe when
    // it is closed.
    c = conn{Conn: c, wg: &this.wg}
    
    return
}

