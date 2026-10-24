package mathapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func fixture(t *testing.T) Content {
	t.Helper()
	c, err := LoadContent("../../../content")
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func sessionFixture(t *testing.T, modules ...string) Session {
	t.Helper()
	s, err := NewSession(fixture(t), modules, 0, "test-start-key")
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func answerNext(t *testing.T, s *Session, correct bool) {
	t.Helper()
	q := s.Queue[len(s.Responses)]
	answer := q.Answer
	if !correct {
		answer = "999"
		if q.Kind == "choice" {
			for _, o := range q.Options {
				if o.ID != q.Answer {
					answer = o.ID
					break
				}
			}
		} else if q.Kind == "fraction" {
			answer = "999/1000"
		}
	}
	if err := s.Submit(Response{QuestionID: q.ID, Answer: answer}); err != nil {
		t.Fatal(err)
	}
}
func findEvidence(t *testing.T, s Session, id string) Evidence {
	t.Helper()
	for _, e := range s.Result().Skills {
		if e.Skill.ID == id {
			return e
		}
	}
	t.Fatal("missing evidence")
	return Evidence{}
}

func TestExactGrading(t *testing.T) {
	for _, tc := range []struct {
		q                Question
		input            string
		correct, invalid bool
	}{
		{Question{Kind: "number", Answer: "0.3"}, "0.300", true, false},
		{Question{Kind: "number", Answer: "0.3"}, "０．３", true, false},
		{Question{Kind: "number", Answer: "-0.125"}, "−0.125", true, false},
		{Question{Kind: "number", Answer: "0.3"}, "0.30000000000000001", false, false},
		{Question{Kind: "number", Answer: "0.3"}, "3e-1", false, true},
		{Question{Kind: "number", Answer: "0.3"}, "0.3千克", false, true},
		{Question{Kind: "fraction", Answer: "2/3"}, "4/6", true, false},
		{Question{Kind: "fraction", Answer: "2/3"}, "2/0", false, true},
		{Question{Kind: "fraction", Answer: "4/6", RequiredDenominator: 6}, "2/3", false, false},
		{Question{Kind: "fraction", Answer: "4/6", RequiredDenominator: 6}, "４／６", true, false},
		{Question{Kind: "choice", Answer: "a", Options: []Option{{ID: "a"}, {ID: "b"}}}, "c", false, true},
	} {
		t.Run(tc.q.Kind+tc.input, func(t *testing.T) {
			correct, err := Grade(tc.q, tc.input)
			if (err != nil) != tc.invalid || correct != tc.correct {
				t.Fatalf("correct=%v err=%v", correct, err)
			}
		})
	}
}

func TestQuestionAnswersIndependentlyRecalculated(t *testing.T) {
	// Fixed answer keys are recalculated independently of the generator. This
	// checks mathematical answers, not a claim of teacher/field validation.
	expected := map[string][]string{
		"decimal-compare": {"gt", "lt", "eq", "lt", "gt", "lt"}, "decimal-scale": {"34.5", "0.86", "72", "0.456", "30.6", "2.74"},
		"fraction-equivalent": {"4/6", "9/15", "3/12", "8/14", "6/16", "15/18"}, "fraction-compare": {"lt", "gt", "eq", "lt", "gt", "lt"},
		"factor-identify": {"6", "5", "3", "8", "6", "9"}, "factor-common": {"12", "24", "18", "10", "24", "20"},
		"algebra-expression": {"correct", "correct", "correct", "correct", "correct", "correct"}, "algebra-equation": {"12", "16", "15", "19", "19", "23"},
		"geometry-area": {"20", "24", "30", "36", "30", "42"}, "geometry-volume": {"30", "36", "48", "48", "42", "60"},
		"data-chart": {"6", "6", "8", "6", "9", "12"}, "data-chance": {"red", "blue", "equal", "red", "blue", "red"},
	}
	c := fixture(t)
	if len(c.Questions) != 72 {
		t.Fatal("unexpected inventory")
	}
	for _, q := range c.Questions {
		var i int
		if _, err := fmt.Sscanf(strings.TrimPrefix(q.ID, q.SkillID+"-"), "%d", &i); err != nil {
			t.Fatal(err)
		}
		answer := expected[q.SkillID][i-1]
		if correct, err := Grade(q, answer); err != nil || !correct {
			t.Errorf("%s: expected %s, stored %s", q.ID, answer, q.Answer)
		}
	}
}

func TestScreenBudgetAndEvidence(t *testing.T) {
	s := sessionFixture(t, "decimal", "fraction", "factor", "algebra", "geometry", "data")
	answerNext(t, &s, false)
	if e := findEvidence(t, s, "decimal-compare"); e.Status != "confirm" || e.Wrong != 1 {
		t.Fatalf("single wrong overdiagnosed: %+v", e)
	}
	for !s.Done() {
		answerNext(t, &s, false)
	}
	if len(s.Queue) != 10 || !s.ScreenEnded {
		t.Fatalf("budget: %d", len(s.Queue))
	}
	if e := findEvidence(t, s, "decimal-compare"); e.Status != "practice" || e.Wrong != 2 {
		t.Fatalf("confirmed wrong: %+v", e)
	}
	if e := findEvidence(t, s, "geometry-area"); e.Status != "confirm" {
		t.Fatalf("unconfirmed module generalized: %+v", e)
	}
	if e := findEvidence(t, s, "decimal-scale"); e.Status != "untested" {
		t.Fatal("untested sibling inferred")
	}
	if s.Result().Tested != 6 || s.Result().Unseen != 6 {
		t.Fatalf("coverage: %+v", s.Result())
	}
}
func TestSkippedIsNotWrongOrTested(t *testing.T) {
	for _, skip := range []string{"not_learned", "dont_know"} {
		t.Run(skip, func(t *testing.T) {
			s := sessionFixture(t, "decimal")
			for !s.Done() {
				q := s.Queue[len(s.Responses)]
				if err := s.Submit(Response{QuestionID: q.ID, Skip: skip}); err != nil {
					t.Fatal(err)
				}
			}
			e := findEvidence(t, s, "decimal-compare")
			if e.Wrong != 0 || e.Correct != 0 || e.Status == "practice" || s.Result().Tested != 0 {
				t.Fatalf("skip misclassified: %+v", e)
			}
			if skip == "not_learned" && len(s.Queue) != 1 {
				t.Fatal("unlearned was probed")
			}
		})
	}
}
func TestMixedEvidenceAndRotation(t *testing.T) {
	s := sessionFixture(t, "decimal")
	answerNext(t, &s, false)
	answerNext(t, &s, true)
	if e := findEvidence(t, s, "decimal-compare"); e.Status != "uncertain" {
		t.Fatal(e)
	}
	next, err := NewSession(fixture(t), []string{"decimal"}, 1, "next-start-key")
	if err != nil {
		t.Fatal(err)
	}
	if next.Queue[0].SkillID != "decimal-scale" {
		t.Fatal("skill rotation failed")
	}
}
func TestLearningRoundSeparatesEvidenceAndUsesNewQuestions(t *testing.T) {
	s := sessionFixture(t, "decimal")
	for !s.Done() {
		answerNext(t, &s, false)
	}
	if err := s.FollowUp("decimal-compare"); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, q := range s.Queue {
		if seen[q.ID] {
			t.Fatal("repeated question")
		}
		seen[q.ID] = true
	}
	answerNext(t, &s, true)
	view := s.View()
	if view.Feedback == nil || view.Phase != "practice" {
		t.Fatal("missing practice feedback")
	}
	answerNext(t, &s, true)
	if s.View().Phase != "retest" {
		t.Fatal("missing retest phase")
	}
	answerNext(t, &s, true)
	if s.View().Feedback != nil {
		t.Fatal("retest answer leaked before completion")
	}
	answerNext(t, &s, true)
	e := findEvidence(t, s, "decimal-compare")
	if e.Wrong != 2 || e.Correct != 0 || e.RetestCount != 2 || e.RetestCorrect != 2 || !e.Practiced {
		t.Fatalf("practice contaminated baseline: %+v", e)
	}
	if err := s.FollowUp("decimal-compare"); err == nil {
		t.Fatal("allowed repeat stock")
	}
}

func TestAllTwelveSkillsCanCompleteLearningLoop(t *testing.T) {
	s := sessionFixture(t, "decimal", "fraction", "factor", "algebra", "geometry", "data")
	for !s.Done() {
		answerNext(t, &s, true)
	}
	for _, skill := range s.Content.Skills {
		if err := s.FollowUp(skill.ID); err != nil {
			t.Fatalf("%s: %v", skill.ID, err)
		}
		for !s.Done() {
			answerNext(t, &s, true)
		}
		e := findEvidence(t, s, skill.ID)
		if e.RetestCount != 2 || e.RetestCorrect != 2 || s.View().CompletedSkill != skill.ID {
			t.Fatalf("loop incomplete: %+v", e)
		}
	}
}
func TestContentValidationRejectsBadReferencesAndCycles(t *testing.T) {
	for _, mutate := range []func(*Content){
		func(c *Content) { c.Questions[0].SkillID = "missing" },
		func(c *Content) { c.Questions[0].Options = nil },
		func(c *Content) {
			c.Graph.Routes[0].Prerequisites = append(c.Graph.Routes[0].Prerequisites, Edge{TopicID: c.Graph.Routes[0].TopicID, PrerequisiteID: c.Graph.Routes[0].TopicID, Strength: "hard"})
		},
		func(c *Content) {
			r := &c.Graph.Routes[0]
			p := r.Prerequisites[0].PrerequisiteID
			r.Unlocks = append(r.Unlocks, Edge{TopicID: p, PrerequisiteID: r.TopicID, Strength: "hard"})
		},
		func(c *Content) { c.Questions[12].RequiredDenominator = 100 },
	} {
		c := fixture(t)
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal("bad content accepted")
		}
	}
}

func TestHTTPPersistenceIsolationRetryAndSnapshots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "math.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	a := &App{Content: fixture(t), Store: store, Revision: "test-revision"}
	h := a.Handler()
	request := func(method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		var data []byte
		if body != nil {
			data, _ = json.Marshal(body)
		}
		r := httptest.NewRequest(method, "http://math.test"+path, bytes.NewReader(data))
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	bootstrap := request("GET", "/api/modules", nil, nil)
	cookies := bootstrap.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].Secure {
		t.Fatal("missing profile")
	}
	cookie := cookies[0]
	w := request("POST", "/api/sessions", map[string]any{"modules": []string{"decimal"}, "key": "idempotent-start"}, cookie)
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	var view View
	json.Unmarshal(w.Body.Bytes(), &view)
	if strings.Contains(w.Body.String(), "explanation") || strings.Contains(w.Body.String(), `"answer"`) {
		t.Fatal("answer exposed")
	}
	created := request("POST", "/api/sessions", map[string]any{"modules": []string{"decimal"}, "key": "idempotent-start"}, cookie)
	var again View
	json.Unmarshal(created.Body.Bytes(), &again)
	if view.ID != again.ID {
		t.Fatal("start duplicated")
	}
	if request("GET", "/api/sessions/"+view.ID, nil, nil).Code != 404 {
		t.Fatal("cross-profile session accessible")
	}
	qID := view.Question.ID
	body := map[string]string{"questionId": qID, "answer": "gt", "skip": ""}
	var wg sync.WaitGroup
	statuses := make(chan int, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- request("POST", "/api/sessions/"+view.ID+"/answers", body, cookie).Code
		}()
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != 200 {
			t.Fatal(status)
		}
	}
	saved, err := store.Get(cookie.Value, view.ID)
	if err != nil || len(saved.Responses) != 1 {
		t.Fatalf("retry duplicated: %v", err)
	}
	a.Content.Questions[0].Answer = "lt"
	if saved.Queue[0].Answer != "gt" {
		t.Fatal("snapshot changed")
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	a.Store = store
	restored := request("GET", "/api/sessions/latest", nil, cookie)
	if restored.Code != 200 {
		t.Fatal(restored.Body.String())
	}
	json.Unmarshal(restored.Body.Bytes(), &again)
	if again.ID != view.ID || again.Step != 1 {
		t.Fatal("restart lost progress")
	}
	r := httptest.NewRequest("POST", "http://math.test/api/sessions", strings.NewReader(`{}`))
	r.Header.Set("Origin", "https://other.test")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin write accepted")
	}
}
