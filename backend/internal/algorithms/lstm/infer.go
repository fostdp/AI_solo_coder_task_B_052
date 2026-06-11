package lstm

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type ModelService struct {
	predictor *LSTMPredictor
	mu        sync.Mutex
	once      sync.Once
	modelPath string
}

var (
	globalModel *ModelService
	modelOnce   sync.Once
)

func GetModelService(modelPath string) *ModelService {
	modelOnce.Do(func() {
		globalModel = &ModelService{
			modelPath: modelPath,
		}
		globalModel.loadOrInit()
	})
	return globalModel
}

func (ms *ModelService) loadOrInit() {
	if ms.modelPath != "" {
		data, err := os.ReadFile(ms.modelPath)
		if err == nil {
			var p LSTMPredictor
			if err := json.Unmarshal(data, &p); err == nil {
				ms.predictor = &p
				log.Printf("[LSTM] model loaded from %s", ms.modelPath)
				return
			}
		}
		log.Printf("[LSTM] failed to load model from %s, initializing new model", ms.modelPath)
	}

	ms.predictor = NewLSTMPredictor(8, 32, 1)
	log.Println("[LSTM] new model initialized (8->32->1)")
}

func (ms *ModelService) Predict(input []float64) []float64 {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	return ms.predictor.Forward(input)
}

func (ms *ModelService) ResetState() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.predictor.Reset()
}

func (ms *ModelService) SaveModel(path string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, err := json.Marshal(ms.predictor)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (ms *ModelService) GetPredictor() *LSTMPredictor {
	return ms.predictor
}
