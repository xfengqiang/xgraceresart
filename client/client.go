package main

import (
    "fmt"
    "net/http"
    "io/ioutil"
)

func sendReq1() {
    id := 0
    for {
        req, e := http.NewRequest("GET", "http://127.0.0.1:8888", nil)
        req.Header.Set("Connection", "close")
        req.Close = true
        client := http.Client{}
        res, e := client.Do(req)
        
        if e != nil{
            fmt.Println("err", e)
        }else{
            buf, _ := ioutil.ReadAll(res.Body)
            //            res.Body.Close()
            fmt.Printf("res:%s id:%d\n",string(buf), id)
        }


        id++
    }
}

func sendReq2() {
   
    id := 0
    for {
        res, e := http.Get("http://127.0.0.1:8888")
        if e != nil{
            fmt.Println("err", e)
        }else{
            buf, _ := ioutil.ReadAll(res.Body)
//            res.Body.Close()
            fmt.Printf("res:%s id:%d\n",string(buf), id)
        }
        
        id++
    }
}
func main() {
    sendReq2()
}
