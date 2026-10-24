package mathapp

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Module struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Number      string `json:"number"`
	Description string `json:"description"`
	Symbol      string `json:"symbol"`
}
type Skill struct {
	ID       string `json:"id"`
	ModuleID string `json:"moduleId"`
	Name     string `json:"name"`
	TopicID  string `json:"topicId"`
	Tip      string `json:"tip"`
}
type Option struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Bar struct {
	Label string `json:"label"`
	Value int    `json:"value"`
}
type Question struct {
	ID                  string         `json:"id"`
	SkillID             string         `json:"skillId"`
	Purpose             string         `json:"purpose"`
	Kind                string         `json:"kind"`
	Prompt              string         `json:"prompt"`
	Answer              string         `json:"answer"`
	Explanation         string         `json:"explanation"`
	Options             []Option       `json:"options,omitempty"`
	RequiredDenominator int            `json:"requiredDenominator,omitempty"`
	Unit                string         `json:"unit,omitempty"`
	Figure              map[string]any `json:"figure,omitempty"`
	Chart               []Bar          `json:"chart,omitempty"`
}
type Edge struct {
	TopicID        string `json:"topicId"`
	PrerequisiteID string `json:"prerequisiteId"`
	Strength       string `json:"strength"`
	Reason         string `json:"reason"`
}
type Route struct {
	SkillID       string `json:"skillId"`
	TopicID       string `json:"topicId"`
	Prerequisites []Edge `json:"prerequisites"`
	Unlocks       []Edge `json:"unlocks"`
}
type Node struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SourceName string `json:"sourceName"`
}
type Projection struct {
	TaxonomyVersion      string  `json:"taxonomyVersion"`
	TopicFileSHA256      string  `json:"topicFileSha256"`
	DependencyFileSHA256 string  `json:"dependencyFileSha256"`
	Nodes                []Node  `json:"nodes"`
	Routes               []Route `json:"routes"`
}
type Content struct {
	Modules   []Module   `json:"modules"`
	Skills    []Skill    `json:"skills"`
	Version   string     `json:"version"`
	Questions []Question `json:"questions"`
	Graph     Projection `json:"graph"`
}

func LoadContent(dir string) (Content, error) {
	var c Content
	load := func(name string, target any) error {
		b, err := os.ReadFile(filepath.Join(dir, name+".json"))
		if err != nil {
			return err
		}
		d := json.NewDecoder(strings.NewReader(string(b)))
		d.DisallowUnknownFields()
		return d.Decode(target)
	}
	var skills struct {
		Modules []Module `json:"modules"`
		Skills  []Skill  `json:"skills"`
	}
	var questions struct {
		Version   string     `json:"version"`
		Questions []Question `json:"questions"`
	}
	if err := load("skills", &skills); err != nil {
		return c, err
	}
	if err := load("questions", &questions); err != nil {
		return c, err
	}
	if err := load("taxonomy-map", &c.Graph); err != nil {
		return c, err
	}
	c.Modules, c.Skills, c.Version, c.Questions = skills.Modules, skills.Skills, questions.Version, questions.Questions
	return c, c.Validate()
}

func (c Content) Validate() error {
	if c.Version == "" || len(c.Modules) == 0 || len(c.Skills) == 0 {
		return fmt.Errorf("empty content")
	}
	modules, skills, nodes, questionIDs := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, m := range c.Modules {
		if m.ID == "" || m.Name == "" || modules[m.ID] {
			return fmt.Errorf("invalid module %s", m.ID)
		}
		modules[m.ID] = true
	}
	for _, n := range c.Graph.Nodes {
		if n.ID == "" || nodes[n.ID] {
			return fmt.Errorf("invalid node %s", n.ID)
		}
		nodes[n.ID] = true
	}
	for _, s := range c.Skills {
		if s.ID == "" || s.Name == "" || s.Tip == "" || skills[s.ID] || !modules[s.ModuleID] || !nodes[s.TopicID] {
			return fmt.Errorf("invalid skill %s", s.ID)
		}
		skills[s.ID] = true
	}
	stock := map[string]map[string]int{}
	for _, q := range c.Questions {
		if q.ID == "" || questionIDs[q.ID] || !skills[q.SkillID] || q.Prompt == "" || q.Explanation == "" {
			return fmt.Errorf("invalid question %s", q.ID)
		}
		questionIDs[q.ID] = true
		if !contains([]string{"screen", "confirm", "practice", "retest"}, q.Purpose) {
			return fmt.Errorf("invalid purpose %s", q.ID)
		}
		if stock[q.SkillID] == nil {
			stock[q.SkillID] = map[string]int{}
		}
		stock[q.SkillID][q.Purpose]++
		if q.Kind == "choice" {
			seen, correct := map[string]bool{}, false
			for _, o := range q.Options {
				if o.ID == "" || o.Text == "" || seen[o.ID] {
					return fmt.Errorf("invalid options %s", q.ID)
				}
				seen[o.ID] = true
				correct = correct || o.ID == q.Answer
			}
			if len(q.Options) < 2 || !correct {
				return fmt.Errorf("missing correct option %s", q.ID)
			}
		} else if q.Kind == "number" || q.Kind == "fraction" {
			if _, err := ParseNumber(q.Answer, q.Kind); err != nil {
				return fmt.Errorf("invalid answer %s: %w", q.ID, err)
			}
			if q.RequiredDenominator != 0 && (q.Kind != "fraction" || q.RequiredDenominator < 1) {
				return fmt.Errorf("invalid denominator rule %s", q.ID)
			}
			if q.RequiredDenominator > 0 {
				if correct, err := Grade(q, q.Answer); err != nil || !correct {
					return fmt.Errorf("answer violates rule %s", q.ID)
				}
			}
		} else {
			return fmt.Errorf("unknown kind %s", q.ID)
		}
	}
	for id := range skills {
		for purpose, min := range map[string]int{"screen": 1, "confirm": 1, "practice": 2, "retest": 2} {
			if stock[id][purpose] < min {
				return fmt.Errorf("insufficient %s stock for %s", purpose, id)
			}
		}
	}
	edges, adjacency, routeSkills := map[string]bool{}, map[string][]string{}, map[string]bool{}
	for _, r := range c.Graph.Routes {
		if !skills[r.SkillID] || routeSkills[r.SkillID] {
			return fmt.Errorf("invalid route %s", r.SkillID)
		}
		routeSkills[r.SkillID] = true
		var skill Skill
		for _, s := range c.Skills {
			if s.ID == r.SkillID {
				skill = s
			}
		}
		if r.TopicID != skill.TopicID {
			return fmt.Errorf("route topic mismatch")
		}
		for _, e := range append(append([]Edge{}, r.Prerequisites...), r.Unlocks...) {
			if !nodes[e.TopicID] || !nodes[e.PrerequisiteID] || e.TopicID == e.PrerequisiteID || !contains([]string{"hard", "soft"}, e.Strength) {
				return fmt.Errorf("invalid graph edge")
			}
			key := e.TopicID + ":" + e.PrerequisiteID
			if !edges[key] {
				edges[key] = true
				adjacency[e.TopicID] = append(adjacency[e.TopicID], e.PrerequisiteID)
			}
		}
		for _, e := range r.Prerequisites {
			if e.TopicID != r.TopicID {
				return fmt.Errorf("invalid prerequisite direction")
			}
		}
		for _, e := range r.Unlocks {
			if e.PrerequisiteID != r.TopicID {
				return fmt.Errorf("invalid unlock direction")
			}
		}
	}
	if len(routeSkills) != len(skills) {
		return fmt.Errorf("missing skill routes")
	}
	colors := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		if colors[id] == 1 {
			return false
		}
		if colors[id] == 2 {
			return true
		}
		colors[id] = 1
		for _, p := range adjacency[id] {
			if !visit(p) {
				return false
			}
		}
		colors[id] = 2
		return true
	}
	for id := range nodes {
		if !visit(id) {
			return fmt.Errorf("graph cycle")
		}
	}
	return nil
}

var decimalPattern = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)$`)
var fractionPattern = regexp.MustCompile(`^[+-]?[0-9]+/[0-9]+$`)

func normalize(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		switch r {
		case '／':
			return '/'
		case '．':
			return '.'
		case '－', '−':
			return '-'
		case '＋':
			return '+'
		}
		return r
	}, strings.TrimSpace(s))
}
func ParseNumber(answer, kind string) (*big.Rat, error) {
	answer = normalize(answer)
	if len(answer) > 80 || (kind == "fraction" && !fractionPattern.MatchString(answer)) || (kind == "number" && !decimalPattern.MatchString(answer)) {
		return nil, fmt.Errorf("请输入%s，不要带单位", map[string]string{"fraction": "分数，例如 2/3", "number": "数字，例如 0.25"}[kind])
	}
	n, ok := new(big.Rat).SetString(answer)
	if !ok {
		return nil, fmt.Errorf("分母不能为 0")
	}
	return n, nil
}
func Grade(q Question, answer string) (bool, error) {
	if q.Kind == "choice" {
		for _, o := range q.Options {
			if o.ID == answer {
				return answer == q.Answer, nil
			}
		}
		return false, fmt.Errorf("请选择一个选项")
	}
	actual, err := ParseNumber(answer, q.Kind)
	if err != nil {
		return false, err
	}
	if q.RequiredDenominator > 0 {
		parts := strings.Split(normalize(answer), "/")
		denominator, ok := new(big.Int).SetString(parts[1], 10)
		if !ok || denominator.Cmp(big.NewInt(int64(q.RequiredDenominator))) != 0 {
			return false, nil
		}
	}
	expected, _ := new(big.Rat).SetString(q.Answer)
	return actual.Cmp(expected) == 0, nil
}
func contains(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
}
