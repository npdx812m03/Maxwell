package chain

import (
	"time"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

var m *tf.SavedModel = LoadModel("/home/app/model/", []string{"maxwell"})


func LoadModel(modelPath string, modelNames []string) *tf.SavedModel {
    model, err := tf.LoadSavedModel(modelPath, modelNames, nil)
    if err != nil {
        log.Fatal("LoadSavedModel(): %v", err)
    }

    log.Println("List possible ops in graphs") // print Operator
    for _, op := range model.Graph.Operations() {
        //log.Printf("Op name: %v, on device: %v", op.Name(), op.Device())
        log.Printf("Op name: %v", op.Name())
    }
    return model
}

func GetResource(funcName string, ns []*Nodes, slo int, startTime time.Time) {
    s := m.Session
    state := make([][9]float64, len(ns))
    for i := range ns{
        AppState := ns[i].GetAppState()
        RequestState := []float64{float64(ns[i].Index), 
            float64(slo - time.Since(start).UnixNano() / 1e6), 
            float64(ns[i].GetDescendants())
        }
        Cluster := []float64{
            GetQPS(funcName),
            GetCPU()
        }
        state[i] = append(state[i], AppState...)
        state[i] = append(state[i], RequestState...)
        state[i] = append(state[i], ClusterState...)
    }
    tensor, err := tf.NewTensor(state)

    if err != nil {
        log.Fatal("Error in executing graph...", err)
    }

	result, err := model.Session.Run(
		map[tf.Output]*tf.Tensor{
			model.Graph.Operation("l1").Output(0): tensor,
		},
		[]tf.Output{
			model.Graph.Operation("acts_prob").Output(0),
		}, nil,
	)
	if err != nil {
		fmt.Println(err.Error(),"=======================")
		return
	}
    
	for i := range result{
        ns[i].cpu = randomChoice(result[i].Value) * 10 + ns[i].LeastCPU
    }
}

func randomChoice(p []float64) int{
    var sum float32 = 0.0
    for _, w := range weights {
        sum += w
    }
    r := rand.Float32() * sum
    var t float32 = 0.0
    for i, w := range weights {
        t += w
        if t > r {
            return i
        }
    }
    return len(weights) - 1
}


func UpdateContainer(name string, nums int){
    
}