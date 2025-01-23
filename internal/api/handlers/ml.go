package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/ml"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/validators"
)

type MLHandler struct {
    service *ml.Service
}

func NewMLHandler(service *ml.Service) *MLHandler {
    return &MLHandler{service: service}
}

func (h *MLHandler) GetPrediction(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Symbol    string    `json:"symbol"`
        Features  []float64 `json:"features"`
        ModelName string    `json:"model_name"`
        Version   string    `json:"version,omitempty"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    predReq := &ml.PredictionRequest{
        Symbol:    req.Symbol,
        Features:  req.Features,
        ModelName: req.ModelName,
        Version:   req.Version,
    }

    resp, err := h.service.Predict(r.Context(), predReq)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(resp)
}

func (h *MLHandler) StartTraining(w http.ResponseWriter, r *http.Request) {
    var config ml.TrainingConfig
    if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    jobID, err := h.service.StartTraining(r.Context(), &config)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]int64{"job_id": jobID})
}

func (h *MLHandler) GetTrainingStatus(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    jobID, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid job ID", http.StatusBadRequest)
        return
    }

    status, err := h.service.GetTrainingStatus(r.Context(), jobID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{"status": status})
}

func (h *MLHandler) BatchPredict(w http.ResponseWriter, r *http.Request) {
    var reqs []ml.PredictionRequest
    if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if len(reqs) > 100 {
        http.Error(w, "Maximum batch size exceeded (100)", http.StatusBadRequest)
        return
    }

    var responses []ml.PredictionResponse
    for _, req := range reqs {
        resp, err := h.service.Predict(r.Context(), &req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        responses = append(responses, *resp)
    }

    json.NewEncoder(w).Encode(responses)
}
