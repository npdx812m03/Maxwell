package chain

type DAG struct {
	AdjMap [][]int
	funcSeries []string
	deadline int
}

// type chainStore struct {
// 	AdjMap [][]int
// 	FuncSeries []string
// }

var store map[string]DAG

func InitialDataStore()  {
	store = make(map[string]DAG, 10)
	store["Order_Sync"] = DAG{
		AdjMap: [][]int{{0,1,0,0},{0,0,1,0},{0,0,0,1},{0,0,0,0}},
		// funcSeries: []string{"lstm2389","lstm2389","lstm2389","lstm2389"},
		funcSeries: []string{"fib10","fib10","fib10","fib10"},
		// marker: []int{1, 1, 1, 0},
	}
	store["Batches_Normal"] = DAG{
		AdjMap: [][]int{{0,1,0,0,0},{0,0,1,1,0},{0,0,0,0,1},{0,0,0,0,1},{0,0,0,0,0}},
		funcSeries: []string{"lstm2389","lstm2389","lstm2389","lstm2389","lstm2389"},
		// marker: []int{1, 1, 1, 1, 0},
	}
	store["Hybrid"] = DAG{
		AdjMap: [][]int{{0,1,1,0,0},{0,0,0,1,0},{0,0,0,0,1},{0,0,0,0,1},{0,0,0,0,0}},
		funcSeries: []string{"lstm2389","lstm2389","lstm2389","lstm2389","lstm2389"},
		// marker: []int{1, 1, 0, 1, 0},
	}

	store["Batches_Multiply"] = DAG{
		AdjMap: [][]int{{0,1,1,1,1,0},{0,0,0,0,0,1},{0,0,0,0,0,1},{0,0,0,0,0,1},{0,0,0,0,0,1},{0,0,0,0,0,0}},
		funcSeries: []string{"nodeinfo","nodeinfo","nodeinfo","nodeinfo","nodeinfo","nodeinfo"},
		// marker: []int{1,1,0,0,0,0},
	}

	store["Complex"] = DAG{
		AdjMap: [][]int{{0,1,1,0,0,0,0,0},{0,0,0,0,0,1,0,0},{0,0,0,1,1,0,0,0},{0,0,0,0,0,0,1,0},{0,0,0,0,0,1,1,0},{0,0,0,0,0,0,0,1},{0,0,0,0,0,0,0,1},{0,0,0,0,0,0,0,0}},
		funcSeries: []string{"nodeinfo","nodeinfo","nodeinfo","nodeinfo","nodeinfo","nodeinfo","nodeinfo","nodeinfo"},
	}
	
	store["PictureProcess"] = DAG{
		AdjMap: [][]int{{0,1,0,0,0}, {0,0,1,0,0}, {0,0,0,1,0}, {0,0,0,0,1}, {0,0,0,0,0}},
		funcSeries: [] string{"extract-image-metadata", "transform-metadata", "handler", "thumbnail", "store-image-metadata"},
	}

	store["Profile_lstm2389"] = DAG{
		AdjMap: [][]int{{0}},
		funcSeries: [] string{"lstm2389"},
	}

	store["Profile_bert"] = DAG{
		AdjMap: [][]int{{0}},
		funcSeries: [] string{"bert"},
	}
	store["Profile_catdog"] = DAG{
		AdjMap: [][]int{{0}},
		funcSeries: [] string{"catdog"},
	}
	store["Profile_resnet50"] = DAG{
		AdjMap: [][]int{{0}},
		funcSeries: [] string{"resnet50"},
	}
	store["Face_Extract"] = DAG{
		AdjMap: [][]int{{0,1,0}, {0,0,1}, {0,0,0}},
		funcSeries: []string{"face-extract", "avatar-pre-process", "avatar-generation"},
		deadline: 200
	}

	store["Video_Monitoring"] = DAG{
		AdjMap: [][]int{{0,1,1,0,0}, {0,0,0,0,1}, {0,0,0,1,0}, {0,0,0,0,1}, {0,0,0,0,0}},
		funcSeries: []string{"object-detection", "human-classifier", "vehicle-classifier","license-extraction","data-integration"},
		deadline: 300
	}

	store["Machine-Translate"] = DAG{
		AdjMap: [][]int{{0,1,0,0,0}, {0,0,1,1,0}, {0,0,0,0,1}, {0,0,0,0,1}, {0,0,0,0,0}},
		funcSeries: []string{"text-pre-process", "language-detection", "french-translator", "german-translator", "text-merge"},
		deadline: 150
	}

	store["Speech-Recognition"] = DAG{
		AdjMap: [][]int{{0,1,0,0}, {0,0,1,0}, {0,0,0,1}, {0,0,0,0}},
		funcSeries: []string{"speech-recognition", "text-processing", "neural-translation", "speech-synthesis"},
		deadline: 300
	}
}

func(d *DAG) initialDAG(funcName string)  {
	d.AdjMap = make([][]int, len(store[funcName].AdjMap))
	d.funcSeries = make([]string, len(store[funcName].funcSeries))
	copy(d.AdjMap, store[funcName].AdjMap)
	copy(d.funcSeries, store[funcName].funcSeries)
}