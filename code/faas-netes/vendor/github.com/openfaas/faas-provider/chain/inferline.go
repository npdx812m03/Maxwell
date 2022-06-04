package chain

import (
	"time"
	"list"
)

type Record struct{
	time int
	config int
}
 
type Trace []int
var resource []int = []int{}
var configs map[string][]int = make(map[string][]int)

func init(){
	for i := 0; i < 20; i ++{
		resource = append(resource, 50 + i * 10)
	}
}

func InitializeInferLine(App Job, trace Trace) []int{
	// iterate App Func
	maxQPS := 0
	for i := 0; i < len(trace); i ++{
		if trace[i] > maxQPS{
			maxQPS = trace[i]
		}
	}
	res := []int{}
	for _, node := range App.Nodes{
		node.cpu = BestConfig(node)
		node.PodNum = maxQPS
	} 

	if !Feasible(&App, maxQPS){
		return res
	}
	
	/**	
	for Feasible(&App, maxQPS){
		FindMinThru(App, maxQPS)
		node.PodNum += 1
	}
	**/

	actions := []string{"DowngradeConfig"}
	for true{
		var best *Job
		for _, node := range App.Nodes{
			for _, _ = range actions{
				newJob := DoAction(node, App)
				if Feasible(&newJob, maxQPS) && Cost(&newJob) < Cost(&best){
					best = newJob
				}
			}
		}
		if best != nil{
			App = best
		}else{
			break
		}
	}

	for _, node := range App.Nodes{
		res = append(res, node.cpu)
	} 
	return res
}

func BestConfig(n *Node) int{
	return resource[len(resource) - 1]
}

func DoAction(n *Node, j Job) Job{
	cpu := n.cpu - 10
	for _, newNode := range j.Nodes{
		if newNode.Index == n.Index{
			newNode.cpu = cpu
			break
		}
	}
	return j
}

func Cost(j *Job) int{
	res := 0
	for _, node := range j.Nodes{
		res += node.cpu
	}
	return res
}

func Feasible(j *Job, qps int) bool{
	queue := list.List{}
	for _, node := j.Nodes{
		if len(node.parents) == 0{
			queue.PushBack(node)
		}
	}

	t := 0
	for queue.Len() > 0{
		node := queue.Front().Value.(*Node)
		queue.Remove(queue.Front())
		execTime := GetExeTime(node)
		
		node.endi = node.starti + execTime
		t = Max(node.endi, t)
		node.Ready = 2

		for _, child := node.children{
			allDone := true
			lastEnd := 0
			for _, parent := child.parents{
				if parent.Ready != 2{
					allDone = false
					break
				}
				lastEnd = Max(lastEnd, parent.endi)
			}

			if allDone {
				child.starti = lastEnd
				queue.PushBack(child)
			}
		}

	}
	return j.Deadline >= t
}

func RegulAdjust(){
	for true {
		for name, value := range store{
			job := InitialJob(name, nil, value.deadline)
			configs[name] = InitializeInferLine(job, GetTrace())
		}
		time.Sleep(3 * time.Minute)
	}
}

func GetConfigs(name string){
	return configs[name]
}