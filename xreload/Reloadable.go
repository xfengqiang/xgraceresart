package xreload
import "time"

const (
    STATUS_RUNNING = 0 //running
    STATUS_STOPPING = 1 //stopping
    STATUS_STOPPED = 2 //stopped
    STATUS_STOP_TIMEOUT = 3 //stop timeout
    STATUS_RESTARTING = 4 //restarting
)

type ReloadStatus struct{
    Status int
    Error error
    Msg string
}

func GetStatusStr(s int) string{
    switch s {
        //        case STATUS_RUNNING:
        case STATUS_STOPPING:
        return "stopping"
        case STATUS_STOPPED:
        return "stopped"
        case STATUS_STOP_TIMEOUT:
        return "timeout"
        case STATUS_RESTARTING:
        return "restarting"
    }
    return "running"
}


type Reloadable interface {
    GetId() string
    Stop() error
    Restart() error
    WaitTimeOut(t time.Duration) error
    Status() *ReloadStatus
}



