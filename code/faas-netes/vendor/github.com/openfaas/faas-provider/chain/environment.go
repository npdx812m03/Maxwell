package chain

import (
	"encoding/json"
	"log"
	"sync"
	"strconv"
	"time"
	// "math/rand"
	// "reflect"
	v1app "k8s.io/client-go/listers/apps/v1"
	v1core "k8s.io/client-go/listers/core/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	redis "github.com/go-redis/redis"

)

type Environment struct {
	count int
	Jobs []*Job
	nodeLister v1core.NodeLister
	deploymentLister v1app.DeploymentLister
	CPULeft int
	qps int
	FinishedJobs []int
	Done bool
}

type action struct {
	JobIndex int `json:"job_index"`
	NodeIndex int `json:"node_index"`
	ColdStart bool	`json:"cold_start"`
	CPUAllocate int	`json:"cpu_allocate"`
}


var env *Environment
var index int
var ilock sync.Mutex
var jlock sync.Mutex
var flock sync.Mutex
var client *redis.Client
var readyNodesChannel chan string
var lastInvoke map[string] *time.Time
var maxTime int

func InitialEnv(dlister v1app.DeploymentLister, nlister v1core.NodeLister)  {
	env = new(Environment)
	ilock = sync.Mutex{}
	jlock = sync.Mutex{}
	flock = sync.Mutex{}
	maxTime = 5
	readyNodesChannel = make(chan string)
	lastInvoke = make(map[string] *time.Time)
	env.nodeLister = nlister
	env.deploymentLister = dlister
	env.Jobs = make([]*Job, 0)
	env.CPULeft = getClusterCPULeft()
	env.FinishedJobs = make([]int, 0)
	env.Done = false
	env.count = 0
	
	client = redis.NewClient(&redis.Options{
		Addr:     "192.168.1.120:32190",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

}

func appendJob(j *Job)  {
	jlock.Lock()
	defer jlock.Unlock()
	env.Jobs = append(env.Jobs, j)
	
}

func removeJob(j *Job) {
	for index,item := range env.Jobs {
		if item.Index == j.Index{
			jlock.Lock()
			env.Jobs = append(env.Jobs[:index], env.Jobs[index+1:]...)
			jlock.Unlock()
			break
		}
	}
	flock.Lock()
	env.FinishedJobs = append(env.FinishedJobs, j.Deadline - int(time.Since(j.StartTime).Seconds()*1000))
	log.Println(env.FinishedJobs)
	client.LPush("slo", int(j.Deadline - int(time.Since(j.StartTime).Seconds()*1000)))
	flock.Unlock()

}

func getJobIndex() int{
	ilock.Lock()
	defer ilock.Unlock()
	index += 1
	return index
}

func getClusterCPULeft() int{
	sel := labels.NewSelector()
	req, err := labels.NewRequirement("openfaas-zhy", selection.Equals, []string{"worker"})
	if err != nil {
		return 0
	}
	workers:= sel.Add(*req)

	nodes , err := env.nodeLister.List(workers)
	if err != nil {
		panic(err)
	}

	allocatableCPU := 0
	for _,node := range nodes {
		
		cpu, _ := strconv.Atoi(node.Status.Allocatable.Cpu().String())
		allocatableCPU += cpu
	}
	log.Println(allocatableCPU)
	return allocatableCPU
}

func getPodNum(name string) int{
	pod, _ := env.deploymentLister.Deployments("openfaas-fn-zhy").Get(name)

	return int(pod.Status.AvailableReplicas)
}

func CommunicateWithTrainer() {
	for true{
		_ = <- readyNodesChannel
		state, err := json.Marshal(env)
		for err != nil{
			state, err = json.Marshal(env)
		}
		startTime := time.Now()
		
		client.LPush("env", string(state))
		result, _ := client.BLPop(time.Duration(200)*time.Second, "action").Result()
		act := action{}
		if len(result) >= 1 {
			log.Println(env.count)
			env.count += 1
			err := json.Unmarshal([]byte(string(result[1])), &act)
			if err!=nil{
				log.Println(err)
			}
				for _, job := range env.Jobs{
					if job.Index == act.JobIndex {
						for _, node := range job.Nodes{
							if node.Index == act.NodeIndex {
								node.cpu = act.CPUAllocate
								node.Ready = 2
								node.position.Done()
								(*lastInvoke[node.Name]) = time.Now()
								break
							}
						}
					}
				}
		}

		log.Println("获取决策耗时", time.Since(startTime))
	}
}


// func Reset() {
	// for true{
	// 	res, _ := client.BLPop(time.Duration(200)*time.Second, "reset").Result()
	// 	log.Println(res)
	// 		if len(res) >= 1{
				
	// 		// if maxTime * 2 > 40{
	// 		// 	maxTime = 40
	// 		// }else{
	// 		// 	maxTime = maxTime * 2
	// 		// }
	// 	}
	// }
// }

