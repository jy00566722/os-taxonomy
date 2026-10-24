package mathapp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	randv2 "math/rand/v2"
	"time"
)

type Response struct {
	QuestionID string `json:"questionId"`
	Answer     string `json:"answer"`
	Skip       string `json:"skip,omitempty"`
	Correct    bool   `json:"correct"`
	At         int64  `json:"at"`
}
type Session struct {
	ID               string     `json:"id"`
	Created          int64      `json:"created"`
	Content          Content    `json:"content"`
	Modules          []string   `json:"modules"`
	Queue            []Question `json:"queue"`
	Responses        []Response `json:"responses"`
	ScreenEnded      bool       `json:"screenEnded"`
	ConfirmScheduled bool       `json:"confirmScheduled"`
	RoundStart       int        `json:"roundStart"`
	FollowUpSkills   []string   `json:"followUpSkills"`
	StartKey         string     `json:"startKey"`
}

func token() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func NewSession(content Content, modules []string, rotation int, startKey string) (Session, error) {
	if len(modules) == 0 || len(modules) > len(content.Modules) {
		return Session{}, fmt.Errorf("请至少选择一个学过的模块")
	}
	seen := map[string]bool{}
	for _, id := range modules {
		known := false
		for _, m := range content.Modules {
			if m.ID == id {
				known = true
			}
		}
		if !known || seen[id] {
			return Session{}, fmt.Errorf("模块选择无效")
		}
		seen[id] = true
	}
	s := Session{ID: token(), Created: time.Now().UnixMilli(), Content: content, Modules: modules, Queue: []Question{}, Responses: []Response{}, FollowUpSkills: []string{}, StartKey: startKey}
	for _, m := range content.Modules {
		if !seen[m.ID] {
			continue
		}
		candidates := []Skill{}
		for _, skill := range content.Skills {
			if skill.ModuleID == m.ID {
				candidates = append(candidates, skill)
			}
		}
		if len(candidates) == 0 {
			return Session{}, fmt.Errorf("模块缺少内容")
		}
		skill := candidates[rotation%len(candidates)]
		q, ok := s.available(skill.ID, "screen")
		if !ok {
			return Session{}, fmt.Errorf("缺少筛查题")
		}
		s.Queue = append(s.Queue, shuffled(q))
	}
	return s, nil
}
func shuffled(q Question) Question {
	q.Options = append([]Option(nil), q.Options...)
	randv2.Shuffle(len(q.Options), func(i, j int) { q.Options[i], q.Options[j] = q.Options[j], q.Options[i] })
	return q
}
func (s Session) available(skillID, purpose string) (Question, bool) {
	for _, q := range s.Content.Questions {
		if q.SkillID != skillID || q.Purpose != purpose {
			continue
		}
		used := false
		for _, old := range s.Queue {
			if old.ID == q.ID {
				used = true
				break
			}
		}
		if !used {
			return q, true
		}
	}
	return Question{}, false
}
func (s Session) Done() bool { return len(s.Responses) == len(s.Queue) }
func (s *Session) Submit(input Response) error {
	// Repeated transport delivery returns the saved state even after completion.
	for _, old := range s.Responses {
		if old.QuestionID == input.QuestionID {
			if old.Answer != input.Answer || old.Skip != input.Skip {
				return fmt.Errorf("这道题已提交，请继续下一题")
			}
			return nil
		}
	}
	if s.Done() || s.Queue[len(s.Responses)].ID != input.QuestionID {
		return fmt.Errorf("题目已变化，请刷新后继续")
	}
	q := s.Queue[len(s.Responses)]
	if input.Skip != "" {
		if !contains([]string{"not_learned", "dont_know"}, input.Skip) || input.Answer != "" {
			return fmt.Errorf("跳过方式无效")
		}
	} else {
		correct, err := Grade(q, input.Answer)
		if err != nil {
			return err
		}
		input.Correct = correct
	}
	input.At = time.Now().UnixMilli()
	s.Responses = append(s.Responses, input)
	if s.Done() && !s.ConfirmScheduled {
		s.ConfirmScheduled = true
		candidates := []Question{}
		// Incorrect/unsure first, then correct answers when the budget permits.
		for _, priority := range []bool{true, false} {
			for i, r := range s.Responses {
				if r.Skip == "not_learned" || ((!r.Correct) != priority) {
					continue
				}
				if q, ok := s.available(s.Queue[i].SkillID, "confirm"); ok {
					candidates = append(candidates, shuffled(q))
				}
			}
		}
		if len(candidates) > 4 {
			candidates = candidates[:4]
		}
		s.Queue = append(s.Queue, candidates...)
	}
	if s.Done() && !s.ScreenEnded {
		s.ScreenEnded = true
	}
	return nil
}
func (s *Session) FollowUp(skillID string) error {
	if !s.Done() || !s.ScreenEnded {
		return fmt.Errorf("请先完成当前这一轮")
	}
	// Retries cannot create a second round for the same skill.
	if contains(s.FollowUpSkills, skillID) {
		return fmt.Errorf("这个知识点本轮已练习过，可以开始一次新体检")
	}
	var skill Skill
	for _, candidate := range s.Content.Skills {
		if candidate.ID == skillID {
			skill = candidate
		}
	}
	if skill.ID == "" || !contains(s.Modules, skill.ModuleID) {
		return fmt.Errorf("请选择本次范围内的知识点")
	}
	questions := []Question{}
	for _, purpose := range []string{"practice", "retest"} {
		for _, q := range s.Content.Questions {
			if q.SkillID == skillID && q.Purpose == purpose {
				questions = append(questions, shuffled(q))
			}
		}
	}
	if len(questions) != 4 {
		return fmt.Errorf("练习题暂未准备好")
	}
	s.RoundStart = len(s.Queue)
	s.Queue = append(s.Queue, questions...)
	s.FollowUpSkills = append(s.FollowUpSkills, skillID)
	return nil
}

type PublicQuestion struct {
	ID      string         `json:"id"`
	SkillID string         `json:"skillId"`
	Kind    string         `json:"kind"`
	Prompt  string         `json:"prompt"`
	Purpose string         `json:"purpose"`
	Options []Option       `json:"options,omitempty"`
	Unit    string         `json:"unit,omitempty"`
	Figure  map[string]any `json:"figure,omitempty"`
	Chart   []Bar          `json:"chart,omitempty"`
}

func publicQuestion(q Question) PublicQuestion {
	return PublicQuestion{q.ID, q.SkillID, q.Kind, q.Prompt, q.Purpose, q.Options, q.Unit, q.Figure, q.Chart}
}

type Feedback struct {
	Correct     bool   `json:"correct"`
	Skipped     bool   `json:"skipped"`
	Explanation string `json:"explanation"`
	QuestionID  string `json:"questionId"`
}
type View struct {
	ID             string          `json:"id"`
	Done           bool            `json:"done"`
	Step           int             `json:"step"`
	Total          int             `json:"total"`
	Phase          string          `json:"phase"`
	Version        string          `json:"version"`
	Question       *PublicQuestion `json:"question"`
	Skill          *Skill          `json:"skill"`
	Feedback       *Feedback       `json:"feedback"`
	Modules        []string        `json:"modules"`
	CompletedSkill string          `json:"completedSkill,omitempty"`
}

func (s Session) View() View {
	v := View{ID: s.ID, Done: s.Done(), Step: len(s.Responses) - s.RoundStart, Total: len(s.Queue) - s.RoundStart, Phase: "screen", Version: s.Content.Version, Modules: s.Modules}
	if !s.ConfirmScheduled {
		v.Total += min(4, len(s.Queue))
	}
	if s.Done() && len(s.FollowUpSkills) > 0 {
		v.CompletedSkill = s.FollowUpSkills[len(s.FollowUpSkills)-1]
	}
	if !s.Done() {
		q := s.Queue[len(s.Responses)]
		p := publicQuestion(q)
		v.Question = &p
		v.Phase = q.Purpose
		for _, skill := range s.Content.Skills {
			if skill.ID == q.SkillID {
				copy := skill
				v.Skill = &copy
			}
		}
	}
	if len(s.Responses) > 0 {
		i := len(s.Responses) - 1
		q := s.Queue[i]
		if q.Purpose == "practice" {
			r := s.Responses[i]
			v.Feedback = &Feedback{r.Correct, r.Skip != "", q.Explanation, q.ID}
		}
	}
	return v
}

type Evidence struct {
	Skill         Skill  `json:"skill"`
	Status        string `json:"status"`
	Correct       int    `json:"correct"`
	Wrong         int    `json:"wrong"`
	Skipped       int    `json:"skipped"`
	RetestCorrect int    `json:"retestCorrect"`
	RetestCount   int    `json:"retestCount"`
	Practiced     bool   `json:"practiced"`
}
type Review struct {
	Question    PublicQuestion `json:"question"`
	Answer      string         `json:"answer"`
	Correct     bool           `json:"correct"`
	Skip        string         `json:"skip"`
	Explanation string         `json:"explanation"`
}
type Result struct {
	ID             string     `json:"id"`
	Skills         []Evidence `json:"skills"`
	Recommendation string     `json:"recommendation"`
	Tested         int        `json:"tested"`
	Unseen         int        `json:"unseen"`
	Review         []Review   `json:"review"`
	Created        int64      `json:"created"`
}

func (s Session) Result() Result {
	result := Result{ID: s.ID, Created: s.Created, Skills: []Evidence{}, Review: []Review{}}
	for _, skill := range s.Content.Skills {
		e := Evidence{Skill: skill, Status: "untested"}
		notLearned := 0
		if !contains(s.Modules, skill.ModuleID) {
			e.Status = "outside"
		}
		for i, r := range s.Responses {
			q := s.Queue[i]
			if q.SkillID != skill.ID {
				continue
			}
			switch q.Purpose {
			case "screen", "confirm":
				if r.Skip != "" {
					e.Skipped++
					if r.Skip == "not_learned" {
						notLearned++
					}
				} else if r.Correct {
					e.Correct++
				} else {
					e.Wrong++
				}
			case "practice":
				e.Practiced = true
			case "retest":
				e.RetestCount++
				if r.Correct && r.Skip == "" {
					e.RetestCorrect++
				}
			}
		}
		if e.Correct+e.Wrong+e.Skipped > 0 {
			switch {
			case notLearned > 0 && e.Correct+e.Wrong == 0:
				e.Status = "not_learned"
			case e.Wrong >= 2:
				e.Status = "practice"
			case e.Correct >= 2 && e.Wrong == 0:
				e.Status = "smooth"
			case e.Correct > 0 && e.Wrong > 0:
				e.Status = "uncertain"
			case e.Wrong > 0 || e.Skipped > 0:
				e.Status = "confirm"
			default:
				e.Status = "observed"
			}
		}
		if e.Correct+e.Wrong > 0 {
			result.Tested++
		} else if e.Status != "outside" {
			result.Unseen++
		}
		result.Skills = append(result.Skills, e)
	}
	for _, status := range []string{"practice", "confirm", "uncertain", "observed", "smooth", "untested"} {
		for _, e := range result.Skills {
			if result.Recommendation == "" && e.Status == status && !e.Practiced {
				result.Recommendation = e.Skill.ID
			}
		}
	}
	if s.Done() {
		for i, r := range s.Responses {
			q := s.Queue[i]
			answer := r.Answer
			if q.Kind == "choice" {
				for _, o := range q.Options {
					if o.ID == r.Answer {
						answer = o.Text
					}
				}
			}
			result.Review = append(result.Review, Review{publicQuestion(q), answer, r.Correct, r.Skip, q.Explanation})
		}
	}
	return result
}
