package chain

import (
	"log"
	"net/http"
	"time"
	"sync"
)


type Job struct {
	Index int
	StartTime time.Time
	endTime time.Time
	Nodes []*Node
	StartNodes []*Node
	DAG *DAG
	Deadline int
	funcSeriesName string
	proxyClient *http.Client
	done chan *http.Response

	rlock sync.Mutex
	Resource int
}


func (j *Job)InitialJob(funcName string, proxyClient *http.Client, deadline int)  {
	j.StartTime = time.Now()
	j.Deadline = deadline
	
	j.Index = getJobIndex()

	j.rlock = sync.Mutex{}

	j.Resource = 0

	j.done = make(chan *http.Response, 1)

	j.funcSeriesName = funcName

	j.proxyClient = proxyClient

	j.DAG = &DAG{}
	j.DAG.initialDAG(funcName)

	j.Nodes = make([]*Node, len(j.DAG.AdjMap))

	for i := range j.DAG.AdjMap {
		j.Nodes[i] = new(Node)
		j.Nodes[i].Index = i
		j.Nodes[i].Name = j.DAG.funcSeries[i]
		j.Nodes[i].belongJob = j
		if _, ok := lastInvoke[j.Nodes[i].Name]; !ok {

			t := time.Now() 
			lastInvoke[j.Nodes[i].Name] = &t
		}
		j.Nodes[i].LastInvoke = lastInvoke[j.Nodes[i].Name]
		j.Nodes[i].position.Add(1)
	}

	for i := range j.DAG.AdjMap {
		from := j.Nodes[i]
		for k := range j.DAG.AdjMap[i] {
			if j.DAG.AdjMap[i][k] == 1 {
				to := j.Nodes[k]
				from.children = append(from.children, to)
				to.parents = append(to.parents, from)
				to.info.Add(1)
			}
		}
	}
	for i := range j.Nodes{
		if len(j.Nodes[i].parents) == 0{
			j.StartNodes = append(j.StartNodes, j.Nodes[i])
		}
		filepathNames, _ := filepath.Glob(filepath.Join("/home/app/func_data/" + j.Nodes[i].Name,"*"))
		data, err := ioutil.ReadFile(filepathNames[0])
		j.Nodes[i].avgCPU, _ := strconv.ParseFloat(strings.Split(string(data), ",")[0], 64)
		j.Nodes[i].avgLantency, _ := strconv.ParseFloat(strings.Split(string(data), ",")[1], 64)
	}
	// log.Println(time.Since(j.StartTime))

}


func (j *Job)Start(originalReq *http.Request, resolver BaseURLResolver)  *http.Response{
	j.Nodes[0].request = originalReq
	for _, node := range j.Nodes {
		GetResource(j.funcSeriesName, j.StartNodes, j.Deadline, j.StartTime)
		go node.Invoke(resolver)
	}

	res := <- j.done
	j.endTime = time.Now()
	log.Println(j.endTime)
	log.Println("end", j.Index)
	return res
}