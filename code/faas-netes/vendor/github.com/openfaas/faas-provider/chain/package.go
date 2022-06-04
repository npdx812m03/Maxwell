package chain
import (
	"time"
	"io/ioutil"
	"path/filepath"
	// "fmt"
	"strconv"
	"strings"
)


var apptables map[string][][]float64
var appactions map[string][][][]int

func InitialTables()  {
	apptables = make(map[string][][]float64)
	appactions= make(map[string][][][]int)

	filepathNames, _ := filepath.Glob(filepath.Join("/home/app/latency_data/fib10","*"))
	funcname := make(map[int]int)
	for i := range filepathNames{
		// fmt.Println(filepathNames[i])
		data, err := ioutil.ReadFile(filepathNames[i])
		if err != nil {
			continue
		}
		names := strings.Split(filepathNames[i], "/")
		cpu,_ := strconv.Atoi(strings.Split(names[len(names) - 1], ".")[0])
		latency := 0
		lens := 0
		for _, value := range strings.Split(string(data), "\n"){
			value, err := strconv.ParseFloat(value, 64)
			if err != nil{
				continue
			}
			latency += int(value*1000)
			lens += 1
		}
		latency /= lens
		funcname[cpu] = latency
	}
	deadline := 2000
	dag_len := 4
	table1 := make([][]float64, dag_len)
	action1:= make([][][]int, dag_len)
	for i := 0; i < dag_len; i+=1{
		table1[i] = make([]float64, deadline)
		action1[i] = make([][]int, deadline)
		for j := 0; j < deadline; j++ {
			action1[i][j] = make([]int, dag_len)
		}
	}

	for i := 0; i < deadline; i+=1{
		table1[dag_len - 1][i] = -1
		for key, val := range funcname{
			if val <= i && (float64(100/float64(key)) >= table1[dag_len - 1][i] || table1[dag_len-1][i] == -1){
				table1[dag_len - 1][i] = float64(100/float64(key))
				action1[dag_len - 1][i][dag_len - 1] = key
			}
		}
	}
	// fmt.Println(action1[3])

	for i := dag_len - 2; i >= 0; i-- {
		for j := 0; j < deadline; j++ {
			for key, val := range funcname{
				if val<=j && table1[i+1][j-val]+float64(100/float64(key)) >= table1[i][j] && table1[i+1][j-val]!=-1{
					table1[i][j] = table1[i+1][j - val] + float64(100/float64(key))
					for k := 0 ; k < dag_len; k++{
						action1[i][j][k] = action1[i+1][j - val][k]
					}
					action1[i][j][i] = key
				}
			}
		}
	}
	
	apptables["Order_Sync"] = table1
	appactions["Order_Sync"] = action1
	// fmt.Println(table1)
}

func queryResource(appName string, stage int, slo int, startTime time.Time) int{
	alpha := 0.92
	lastTime := slo - int(time.Since(startTime).Seconds()*1000)
	if lastTime < 0{
		lastTime = 0
	}

	// fmt.Println(lastTime," ",stage)
	actions := appactions[appName]
	act := actions[stage][int(float64(lastTime)*alpha)][stage]
	if act == 0{
		act = 200
	}
	if act % 10 != 0 {
		act += 5
	}
	return act
}