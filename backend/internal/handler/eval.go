package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/auliaafriza/personalgpt-backend/internal/db"
	"github.com/auliaafriza/personalgpt-backend/internal/eval"
	appmw "github.com/auliaafriza/personalgpt-backend/internal/middleware"
)

type EvalHandler struct {
	evalRepo    *db.EvalRepo
	msgRepo     *db.MessageRepo
	convRepo    *db.ConversationRepo
	retrievalEv *eval.RetrievalEvaluator
	judgeEv     *eval.JudgeEvaluator
}

func NewEvalHandler(
	evalRepo *db.EvalRepo,
	msgRepo *db.MessageRepo,
	convRepo *db.ConversationRepo,
	retrievalEv *eval.RetrievalEvaluator,
	judgeEv *eval.JudgeEvaluator,
) *EvalHandler {
	return &EvalHandler{
		evalRepo:    evalRepo,
		msgRepo:     msgRepo,
		convRepo:    convRepo,
		retrievalEv: retrievalEv,
		judgeEv:     judgeEv,
	}
}

// --- Eval sets CRUD ------------------------------------------------------

// GET /eval-sets
func (h *EvalHandler) ListSets(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	sets, err := h.evalRepo.ListSetsByUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list eval sets")
		return
	}
	writeJSON(w, http.StatusOK, sets)
}

type createEvalSetBody struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Queries     []db.EvalSetQuery  `json:"queries"`
}

// POST /eval-sets
func (h *EvalHandler) CreateSet(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body createEvalSetBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	set, err := h.evalRepo.CreateSet(r.Context(), db.CreateEvalSetParams{
		UserID:      user.ID,
		Name:        body.Name,
		Description: body.Description,
		Queries:     body.Queries,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create eval set")
		return
	}
	writeJSON(w, http.StatusCreated, set)
}

// DELETE /eval-sets/{id}
func (h *EvalHandler) DeleteSet(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.evalRepo.DeleteSetByUser(r.Context(), id, user.ID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "eval set not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete eval set")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- Eval runs ----------------------------------------------------------

// GET /eval-runs?kind=&evalSetId=
func (h *EvalHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	q := r.URL.Query()
	runs, err := h.evalRepo.ListRunsByUser(r.Context(), user.ID, db.ListRunsFilter{
		Kind:      q.Get("kind"),
		EvalSetID: q.Get("evalSetId"),
	}, 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

// POST /eval-runs/retrieval — trigger retrieval eval untuk set tertentu.
type retrievalRunBody struct {
	EvalSetID string `json:"evalSetId"`
	TopK      int    `json:"topK"`
}

func (h *EvalHandler) RunRetrievalEval(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body retrievalRunBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.EvalSetID == "" {
		writeError(w, http.StatusBadRequest, "evalSetId is required")
		return
	}
	if body.TopK <= 0 {
		body.TopK = 5
	}

	set, err := h.evalRepo.GetSetByUser(r.Context(), body.EvalSetID, user.ID)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "eval set not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load eval set")
		return
	}

	start := time.Now()
	results := h.retrievalEv.Run(r.Context(), user.ID, set.Queries, body.TopK)
	duration := time.Since(start).Milliseconds()

	// Serialize results to map[string]any so we can put in JSONB.
	resultsMap := map[string]any{
		"topK":         results.TopK,
		"queryCount":   results.QueryCount,
		"avgRecallAtK": results.AvgRecallAtK,
		"avgMRR":       results.AvgMRR,
		"perQuery":     results.PerQuery,
	}
	setID := body.EvalSetID
	run, err := h.evalRepo.CreateRun(r.Context(), db.CreateEvalRunParams{
		UserID:     user.ID,
		Kind:       "retrieval",
		EvalSetID:  &setID,
		Results:    resultsMap,
		DurationMs: duration,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save run")
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// POST /eval-runs/judge — LLM-as-judge untuk 1 assistant message tertentu.
type judgeRunBody struct {
	MessageID string `json:"messageId"`
}

func (h *EvalHandler) RunJudgeEval(w http.ResponseWriter, r *http.Request) {
	user := appmw.UserFromCtx(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body judgeRunBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.MessageID == "" {
		writeError(w, http.StatusBadRequest, "messageId is required")
		return
	}

	// Load message + previous user turn + verify ownership via conversation.
	msg, err := h.findAssistantMessage(r, user.ID, body.MessageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "message not found or not accessible")
		return
	}
	prev, err := h.findPrecedingUserMessage(r, msg.ConversationID, msg.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot find preceding user message")
		return
	}

	start := time.Now()
	results := h.judgeEv.Judge(r.Context(), prev.Content, msg.Content, msg.Sources)
	duration := time.Since(start).Milliseconds()

	resultsMap := map[string]any{
		"model":        results.Model,
		"faithfulness": results.Faithfulness,
		"helpfulness":  results.Helpfulness,
		"reasoning":    results.Reasoning,
	}
	if results.ParsingError != "" {
		resultsMap["parsingError"] = results.ParsingError
		resultsMap["rawResponse"] = results.RawResponse
	}
	msgID := msg.ID
	run, err := h.evalRepo.CreateRun(r.Context(), db.CreateEvalRunParams{
		UserID:           user.ID,
		Kind:             "judge",
		SubjectMessageID: &msgID,
		Results:          resultsMap,
		DurationMs:       duration,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save run")
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// findAssistantMessage looks up assistant message by ID; verifies belongs to user
// by joining via conversation ownership check.
func (h *EvalHandler) findAssistantMessage(r *http.Request, userID, msgID string) (db.Message, error) {
	// Scan through user's conversations. Not ideal for scale but OK for eval.
	convs, err := h.convRepo.ListByUser(r.Context(), userID, 200)
	if err != nil {
		return db.Message{}, err
	}
	for _, c := range convs {
		msgs, err := h.msgRepo.ListByConversation(r.Context(), c.ID)
		if err != nil {
			continue
		}
		for _, m := range msgs {
			if m.ID == msgID && m.Role == db.RoleAssistant {
				return m, nil
			}
		}
	}
	return db.Message{}, errors.New("not found")
}

func (h *EvalHandler) findPrecedingUserMessage(r *http.Request, convID, assistantMsgID string) (db.Message, error) {
	msgs, err := h.msgRepo.ListByConversation(r.Context(), convID)
	if err != nil {
		return db.Message{}, err
	}
	var prev *db.Message
	for i, m := range msgs {
		if m.ID == assistantMsgID {
			// walk back untuk cari terakhir user role
			for j := i - 1; j >= 0; j-- {
				if msgs[j].Role == db.RoleUser {
					prev = &msgs[j]
					break
				}
			}
			break
		}
	}
	if prev == nil {
		return db.Message{}, errors.New("no preceding user message")
	}
	return *prev, nil
}
