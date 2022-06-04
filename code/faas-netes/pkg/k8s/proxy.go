// Copyright (c) Alex Ellis 2017. All rights reserved.
// Copyright 2020 OpenFaaS Author(s)
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package k8s

import (
	"fmt"
	"math/rand" 
	"time"
	// "container/list"
	"net/url"
	"strings"
	"sync"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	types "github.com/openfaas/faas-provider/types"
	chain "github.com/openfaas/faas-provider/chain"
	corelister "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/api/core/v1"
	// v1 "k8s.io/client-go"
)

// watchdogPort for the OpenFaaS function watchdog
const watchdogPort = 8080
// var IdlePod map[string][]string = make(map[string][]string)

var IdlePod map[string]map[int]chan string = make(map[string]map[int]chan string)
var BusyPod map[string]int  = make(map[string]int)
var rlock map[string]sync.Mutex = make(map[string]sync.Mutex)
var IPLock map[string]*sync.Mutex = make(map[string]*sync.Mutex)
var dockers map[string] types.FunctionDockerMetadata = make(map[string] types.FunctionDockerMetadata)
var nodeIPTable map[string] string = make(map[string] string)

var chainFunctions = map[string][]string{}
var initial = make(map[string]int)

func NewFunctionLookup(ns string, lister corelister.EndpointsLister) *FunctionLookup {
	chainFunctions["test"] = []string {"nodeinfo", "nodeinfo", "nodeinfo", "nodeinfo"}
	chainFunctions["image-process"] = [] string{"extract-image-metadata", "transform-metadata", "handler", "thumbnail", "store-image-metadata"}
	nodeIPTable["kube-node-3"] = "http://192.168.1.135"
	nodeIPTable["kube-node-4"] = "http://192.168.1.136"
	nodeIPTable["kube-node-5"] = "http://192.168.1.137"
	// nodeIPTable["kube-node-6"] = "http://192.168.1.138"
	nodeIPTable["kube-node-7"] = "http://192.168.1.141"
	// nodeIPTable["kube-node-8"] = "http://192.168.1.140"
	return &FunctionLookup{
		DefaultNamespace: ns,
		EndpointLister:   lister,
		// key是namespace
		Listers:          map[string]corelister.EndpointsNamespaceLister{},
		lock:             sync.RWMutex{},
		// IP: 			  ,
	}
}


type FunctionLookup struct {
	DefaultNamespace string
	EndpointLister   corelister.EndpointsLister
	Listers          map[string]corelister.EndpointsNamespaceLister
	lock sync.RWMutex
	IP string
}


func Contains(Pods []string, IP string) bool{
	var res = false
	for _, value := range Pods{
		if value == IP{
			res = true
			break
		}
	}
	return res
}


func (f *FunctionLookup)GetChain(chainName string) []string{
	if _, ok:= chainFunctions[chainName]; !ok {
		return []string{}
	}
	return chainFunctions[chainName]
}




func InitIdle(functionName string, address []v1.EndpointAddress, pattern string){
	if _, OK := rlock[functionName]; !OK {
		rlock[functionName] = sync.Mutex{}
	}
	var tmp_lock = (rlock[functionName])
	tmp_lock.Lock()
	defer tmp_lock.Unlock()
	client := &http.Client{Timeout: 20 * time.Second}
	nrmap := make(map[string]int)

	if pattern == "Maxwell"{
		for _, value := range nodeIPTable {
			url := value + ":9120/get"
			jsons := "{\"Name\" :\"" + functionName +"\"}"
			req, _ := http.NewRequest("Get", url, bytes.NewBuffer([]byte(jsons)))
			resp, _ := client.Do(req)
			fmt.Println(resp)
			if resp.StatusCode == 200{
				body, _ := ioutil.ReadAll(resp.Body)
				var instances interface{}
				json.Unmarshal(body, &instances)
				for k, v := range instances.(map[string]interface{}){
					nrmap[k] = int(v.(float64)/1000)
				}
			}
			
		}
	}
	

	IdlePod[functionName] = make(map[int]chan string)
	for _, value := range address {

		var dockerInform = value.TargetRef

		var docker = types.FunctionDockerMetadata {
			IP: value.IP,
			NodeIP: nodeIPTable[string(*(value.NodeName))],
			UID: string(dockerInform.UID),
			PodName: string(dockerInform.Name),
			FunctionName: functionName,
			Namespace: dockerInform.Namespace,
			Lock: sync.RWMutex{},
		}
		dockers[value.IP] = docker

		IPLock[value.IP] = new(sync.Mutex)

		if pattern == "Maxwell"{
			if _, ok := IdlePod[functionName][nrmap[docker.PodName]]; !ok{
				IdlePod[functionName][nrmap[docker.PodName]]= make(chan string, len(address))
			}
	
			IdlePod[functionName][nrmap[docker.PodName]] <- value.IP
		}else{
			if _, ok := IdlePod[functionName][-1]; !ok{
				IdlePod[functionName][-1]= make(chan string, len(address))
			}
	
			IdlePod[functionName][-1] <- value.IP
		}
		
	}
}

func AllocatePod(functionName string, CPU int) string{
	var tmp_lock = rlock[functionName]
	tmp_lock.Lock()
	var allocated string

	fmt.Println("等待分配")
	rand.Seed(time.Now().UnixNano())
	fmt.Println(CPU)
	allocated = <- IdlePod[functionName][CPU]
	tmp_lock.Unlock()
	fmt.Println("OK")
	return allocated
}

func (f *FunctionLookup)ReturnPod(functionName string, serviceIP string, CPU int){
	IdlePod[functionName][CPU] <- serviceIP
}


func (f *FunctionLookup) GetLister(ns string) corelister.EndpointsNamespaceLister {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.Listers[ns]
}

func (f *FunctionLookup) SetLister(ns string, lister corelister.EndpointsNamespaceLister) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.Listers[ns] = lister
}

func getNamespace(name, defaultNamespace string) string {
	namespace := defaultNamespace
	if strings.Contains(name, ".") {
		// 返回字符串str中的任何一个字符在字符串s中最后一次出现的位置 hello-world.openfaas-fn-zhy
		namespace = name[strings.LastIndexAny(name, ".")+1:]
	}
	return namespace
}

// 获取对应资源
func (f *FunctionLookup) GetResource(job *chain.Job, stage int) int{
	return 1
}


func (l *FunctionLookup) Resolve(name string, CPU int) (url.URL, types.FunctionDockerMetadata ,error) {
	functionName := name
	namespace := getNamespace(name, l.DefaultNamespace)

	pattern := "Maxwell"

	if err := l.verifyNamespace(namespace); err != nil {
		return url.URL{}, types.FunctionDockerMetadata{}, err
	}

	if strings.Contains(name, ".") {
		// 删除字符串后缀
		functionName = strings.TrimSuffix(name, "."+namespace)
	}

	nsEndpointLister := l.GetLister(namespace)
	

	if nsEndpointLister == nil {
		l.SetLister(namespace, l.EndpointLister.Endpoints(namespace))

		nsEndpointLister = l.GetLister(namespace)
	}

	if pattern != "Maxwell"{
		CPU = -1
	}

	svc, err := nsEndpointLister.Get(functionName)
	if err != nil {
		return url.URL{}, types.FunctionDockerMetadata{}, fmt.Errorf("error listing \"%s.%s\": %s", functionName, namespace, err.Error())
	}

	if len(svc.Subsets) == 0 {
		return url.URL{}, types.FunctionDockerMetadata{}, fmt.Errorf("no subsets available for \"%s.%s\"", functionName, namespace)
	}

	// Pod的虚拟ip
	// log.Println(svc)
	// all := len(svc.Subsets[0].Addresses)
	// log.Printf("svc has all: %d", all)

	if len(svc.Subsets[0].Addresses) == 0 {
		return url.URL{}, types.FunctionDockerMetadata{}, fmt.Errorf("no addresses in subset for \"%s.%s\"", functionName, namespace)
	}

	if ok  := initial[functionName]; ok != len(svc.Subsets[0].Addresses){
		InitIdle(functionName, svc.Subsets[0].Addresses, pattern)
		initial[functionName] = len(svc.Subsets[0].Addresses)
	}

	// 随机选择一个Pod的IP
	// target := rand.Intn(all)

	// serviceIP := svc.Subsets[0].Addresses[target].IP

	serviceIP := AllocatePod(functionName, CPU)

	l.IP = serviceIP
	urlStr := fmt.Sprintf("http://%s:%d", serviceIP, watchdogPort)
	urlRes, err := url.Parse(urlStr)
	if err != nil {
		return url.URL{}, types.FunctionDockerMetadata{}, err
	}

	return *urlRes, dockers[serviceIP], nil
}

func (l *FunctionLookup) verifyNamespace(name string) error {
	if name != "kube-system" {
		return nil
	}
	// ToDo use global namepace parse and validation
	return fmt.Errorf("namespace not allowed")
}

func (l *FunctionLookup) ColdStart(name string) (url.URL, types.FunctionDockerMetadata ,error) {
	return url.URL{}, types.FunctionDockerMetadata{}, nil
}