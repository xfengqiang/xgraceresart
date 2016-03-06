package task
import (
    "time"
    "fmt"
    "errors"
    "sync/atomic"
    "xgracerestart/xreload"
)

type Task struct {
    Id int32 
}

var taskChan chan  *Task
var taskFinished chan  bool
var taskNum int32
var taskReloadable *TaskReloadable

type TaskReloadable struct {
  status int
}

func NewTaskReloadable() *TaskReloadable{
    if taskReloadable == nil {
        taskReloadable = &TaskReloadable{}
        
        taskChan = make(chan  *Task, 100)
        taskFinished  = make(chan bool)
        go ProcessTasks()
    }
   
    return taskReloadable
}

func AddTask(task *Task) error{
    if taskReloadable == nil {
        return errors.New("taskReloadable not inited ")
    }
    if taskReloadable.status != xreload.STATUS_RUNNING {
        return errors.New("taskReloadable is not on runing status ")
    }
    atomic.AddInt32(&taskNum, int32(1))
    taskChan<-task
    return nil
}

func ProcessTasks() {
    for t := range taskChan{
        time.Sleep(2*time.Second)
        fmt.Printf("Processing task:%d\n", t.Id)
        atomic.AddInt32(&taskNum, int32(-1))
    }
}

func (this *TaskReloadable)GetId() string{
    return "db_update_task"
}

func (this *TaskReloadable)Stop() error{
    close(taskChan)
    this.status = xreload.STATUS_STOPPING
    return nil
}

func(this *TaskReloadable)Restart() error{
    close(taskChan)
    this.status = xreload.STATUS_RESTARTING
    return nil
}

func(this *TaskReloadable)WaitTimeOut(t time.Duration) error{
    select {
    case <-taskFinished:
        this.status = xreload.STATUS_STOPPED
    case <-time.After(t):
        this.status = xreload.STATUS_STOP_TIMEOUT
    }
    return nil
}

func (this *TaskReloadable)Status() *xreload.ReloadStatus   {
    info := &xreload.ReloadStatus{Error:nil, Status:this.status}
    switch this.status {
        case xreload.STATUS_STOPPED:
            info.Msg = fmt.Sprintf("Listner task stoped.")
        case xreload.STATUS_STOP_TIMEOUT:
            info.Error = errors.New(fmt.Sprintf("Listner stop time out.Task Left:%d",taskNum))
        default:
        info.Msg = fmt.Sprintf("Listner task running")
    }
    return info
}