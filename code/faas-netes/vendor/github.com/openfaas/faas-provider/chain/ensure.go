package chain

import {
	"time"
	"sync"
}

var coulock sync.Mutex = sync.Mutex{}
var violock sync.Mutex = sync.Mutex{}
var count int
var vio   int
var standard int = 0.14
var extent = 0.20
var NodeIPTable []string = []string{
	"http://192.168.1.135:9120",
	"http://192.168.1.136:9120",
	"http://192.168.1.137:9120",
	"http://192.168.1.138:9120",
	"http://192.168.1.140:9120",
	"http://192.168.1.141:9120",
}


func addNewJob(){
	coulock.Lock()
	defer coulock.Unlock()
	count += 1
}

func removeEnsureJob(job *Job){
	
	if job.endTime.Since(job.StartTime).Millisecond() > job.Deadline{
		violock.Lock()
		vio += 1
		violock.Unlock()
	}

	if vio / count >= standard{
		updateContainerResource()
		coulock.Lock()
		count = 0 
		coulock.Unlock()
		violock.Lock()
		vio = 0
		violock.Unlock()
	}
}

func updateContainerResource(){

	for _, value := range NodeIPTable{
		go func(){

			url := value + "/update"
			jsons := "{\"extent\" : \"" + str(extent) + "\"}"

			req, _ := http.NewRequest("Get", url, bytes.NewBuffer([]byte(jsons)))
			resp, _ := client.Do(req)
		}()
	}
}