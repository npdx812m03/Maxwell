package chain

import (
	"time"
	"sync"
)

var lock sync.Mutex = sync.Mutex{}
var QPSRecord map[string]map[int64]int = make(map[string]map[int64]int)
var maxQ map[string]int = make(map[string]int)
type ExpMovAvg struct{
	shadow int
	decay float64
	times int
}

var ewmaModel map[string]ExpMovAvg = make(map[string]ExpMovAvg)

func(m *ewmaModel) Update(data int) int{
	if m.times < 1{
		m.shadow = data
        m.times += 1
        return data
	}else{
		m.shadow = m.decay * m.shadow + (1 - m.decay) * data
		num := m.shadow / (1 - Pow(m.decay, m.times))
		m.times += 1
		return num
	}
}


func watch(appName string) {
	lock.Lock()
	defer lock.Unlock()
	if QPSRecord[appName] == nil{
		QPSRecord[appName] = make(map[int64]int)
		maxQ[appName] = 0
		ewmaModel[appName] = ExpMovAvg{
			shadow: 1,
			decay: 0.9,
			times: 0
		}
	}
	QPSRecord[appName][time.Now().Unix()] += 1
}

func GetTrace(appName string) []int{
	res := make([]int, 180)
	for i := 179; i >= 0; i -- {
		res[i] = QPSRecord[appName][time.Now().Unix() - int64(i)]
	}
	QPSRecord[appName] = make(map[int64]int)
	return res
}


func GetQPS(appName string) []int{
	lock.Lock()
	defer lock.Unlock()
	if QPSRecord[appName] == nil{
		return 0
	}
	return QPSRecord[appName][time.Now().Unix()]
}

func EWMA() {
	for true{
		for i := range maxQ{
			m := maxQ[i]
			for i := 0; i < 10; i++{
				m = Max(m, QPSRecord[i][time.Now().Unix() - int64(i)])
			}
			nums := ewmaModel[i].Update(m)
			UpdateContainer(i, nums)
		}
		
	}
}

func Pow(i, j int) int{
	num := 1
	for k := 0; k < j; k ++{
		num *= i
	}
	return num
}

func Max(i, j int) int{
	if i > j {
		return i
	}
	return j
}