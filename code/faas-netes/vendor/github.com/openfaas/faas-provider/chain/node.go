package chain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"time"
	"container/list"
	"net/http"
	"net/url"
	"strings"
	"github.com/gorilla/mux"
	"github.com/openfaas/faas-provider/httputil"
	"github.com/openfaas/faas-provider/types"
)

type Node struct {
	Index int
	Name string

	PodNum int
	request *http.Request
	response *http.Response
	output string
	startTime time.Time
	LastInvoke *time.Time

	parents []*Node
	children []*Node
	belongJob *Job

	startMethod bool
	cpu int

	position sync.WaitGroup
	info sync.WaitGroup
	Ready int

	starti int
	endi int

	avgCPU int
	avgLantency
}

type ProfileInform struct{
	FuncName string
	Profile float64
	Resource int
}

const(
	weight = 0.81429
	bias = -1.2373
)

func (n *Node)Invoke(resolver BaseURLResolver) {
	n.info.Wait()
	n.Ready = 1
	// readyNodesChannel <- n
	n.PodNum = getPodNum(n.Name)

	// state, _ := json.Marshal(n.belongJob)
	
	// readyNodesChannel <- string(state)
	
	// n.position.Wait()

	// queryStartTime := time.Now()
	go GetResource(n.belongJob.funcSeriesName, n.children, n.belongJob.Deadline, n.belongJob.StartTime)

	// queryDuration := time.Since(queryStartTime)


	n.startTime = time.Now()

	reader := bytes.NewReader(input)
	n.request, _ = http.NewRequest("POST", "", reader)

	var functionAddr url.URL
	var docker types.FunctionDockerMetadata
	var resolveErr error

	// resolveTime:= time.Now()
	functionAddr, docker, resolveErr = resolver.Resolve(n.Name, n.cpu)
	// log.Println(docker)
	// resolveDuration := time.Since(resolveTime)


	n.belongJob.rlock.Lock()
	n.belongJob.Resource += n.cpu
	n.belongJob.rlock.Unlock()


	if resolveErr != nil {
		log.Printf("resolver ersror: no endpoints for %s\n", n.Name)
		// httputil.Errorf(w, http.StatusServiceUnavailable, "No endpoints available for: %s.", n.Name)
		return
	}

	proxyReq, _ := buildProxyRequest(n.request, functionAddr, mux.Vars(n.request)["params"])

	// reqStartTime := time.Now()
	n.response, _ = n.belongJob.proxyClient.Do(proxyReq.WithContext(n.request.Context()))
	// reqDuration := time.Since(reqStartTime)


	serviceIP := strings.Split(functionAddr.Host, ":")[0]
	resolver.ReturnPod(n.Name, serviceIP, n.cpu)

	n.belongJob.rlock.Lock()
	n.belongJob.Resource -= n.cpu
	n.belongJob.rlock.Unlock()

	n.Ready = 3

	if len(n.children) == 0 {
		n.belongJob.done <- n.response
		return
	}

	body, _ := ioutil.ReadAll(n.response.Body)

	n.output = string(body)

	for _, v := range n.children {
		v.info.Done()
	}
	
}

func ImitateRequest() *http.Response {
	output := fmt.Sprintf("{\"%s\" : \"1\"}", "coooool")
	response := new(http.Response)
	response.Body = ioutil.NopCloser(bytes.NewBuffer([]byte(output)))
	return response
}

func (n *Node) combineInput() {
	var input []byte
	if n.request.Body != nil{
		input, _ = ioutil.ReadAll(n.request.Body)
	}

	for _, v := range n.parents{
		body := []byte(v.output)
		input = combine(body, input)
	}

	c, _ := http.NewRequest("POST", "", nil)
	c.Body = ioutil.NopCloser(bytes.NewBuffer(input))
	n.request = c
}

func combine(output []byte, input []byte) []byte{
	var op map[string]interface{}
	var ip map[string]interface{}

	bop := output
	iop := input

	_ = json.Unmarshal(bop, &op)

	_ = json.Unmarshal(iop, &ip)

	res, _ := json.Marshal(httputil.JsonMerge(op, ip))

	return []byte(string(res))
}

func (n *Node) GetDescendants() int{
	record := make(map[int]int)
	queue := list.List{}
	queue.PushBack(n)
	for queue.Len() > 0{
		node := queue.Front().Value.(*Node)
		queue.Remove(queue.Front())
		for i := range node.children{
			if record[node.children[i].Index] < 1{
				record[node.children[i].Index] = 1
				queue.PushBack(node.children)
			}
		}
	}
	return len(record)
}

func (n *Node) GetAppState() []float64{
	y := []float64{0, 0}
	for i := n.children{
		yy := n.children[i].GetAppState()
		y[0] += yy[0]
		y[1] += yy[1]
	}
	res := concate(relu(weight, bias, y), []float64{n.avgCPU, n.avgLantency})
	return res
}

func relu(w, b float64, y []float64) []float64{
	for i := 0; i < len(y); i++{
		y[i] = w * y[i] + b
		if y[i] < 0{
			y[i] = 0
		}
	}
	return y
}

func concate(y, x []float64) []float64{
	for i := 0; i < len(x); i++{
		x[i] += y[i]
	}

	return x
}