package mathapp

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type App struct {
	Content       Content
	Store         *Store
	WebDir        string
	SecureCookies bool
	Revision      string
	BuiltAt       string
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fail(w, 415, "请求需要 JSON 格式")
		return false
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		fail(w, 400, "请求内容无效")
		return false
	}
	if d.Decode(new(any)) != io.EOF {
		fail(w, 400, "请求内容无效")
		return false
	}
	return true
}
func (a *App) owner(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("math_profile")
	if err == nil && len(cookie.Value) == 48 {
		valid := true
		for _, c := range cookie.Value {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				valid = false
			}
		}
		if valid {
			return cookie.Value
		}
	}
	id := token()
	http.SetCookie(w, &http.Cookie{Name: "math_profile", Value: id, Path: "/api", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: a.SecureCookies || r.TLS != nil, MaxAge: 365 * 24 * 3600})
	return id
}
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]string{"status": "ok", "contentVersion": a.Content.Version, "revision": a.Revision, "builtAt": a.BuiltAt})
	})
	mux.HandleFunc("GET /api/modules", func(w http.ResponseWriter, r *http.Request) {
		a.owner(w, r)
		jsonResponse(w, 200, map[string]any{"modules": a.Content.Modules, "skills": a.Content.Skills, "version": a.Content.Version})
	})
	mux.HandleFunc("POST /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Modules []string `json:"modules"`
			Key     string   `json:"key"`
		}
		if !decode(w, r, &input) {
			return
		}
		if len(input.Key) < 8 || len(input.Key) > 80 {
			fail(w, 400, "开始标识无效，请刷新后重试")
			return
		}
		s, err := a.Store.Start(a.owner(w, r), a.Content, input.Modules, input.Key)
		if err != nil {
			a.storeError(w, err)
			return
		}
		jsonResponse(w, 201, s.View())
	})
	mux.HandleFunc("GET /api/sessions/latest", func(w http.ResponseWriter, r *http.Request) {
		s, err := a.Store.Latest(a.owner(w, r))
		if errors.Is(err, ErrNotFound) {
			jsonResponse(w, 200, nil)
			return
		}
		if err != nil {
			a.storeError(w, err)
			return
		}
		jsonResponse(w, 200, s.View())
	})
	mux.HandleFunc("GET /api/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		s, err := a.Store.Get(a.owner(w, r), r.PathValue("id"))
		if err != nil {
			a.storeError(w, err)
			return
		}
		jsonResponse(w, 200, s.View())
	})
	mux.HandleFunc("POST /api/sessions/{id}/answers", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			QuestionID string `json:"questionId"`
			Answer     string `json:"answer"`
			Skip       string `json:"skip"`
		}
		if !decode(w, r, &input) {
			return
		}
		s, err := a.Store.Update(a.owner(w, r), r.PathValue("id"), func(s *Session) error {
			return s.Submit(Response{QuestionID: input.QuestionID, Answer: input.Answer, Skip: input.Skip})
		})
		if err != nil {
			a.storeError(w, err)
			return
		}
		jsonResponse(w, 200, s.View())
	})
	mux.HandleFunc("GET /api/sessions/{id}/result", func(w http.ResponseWriter, r *http.Request) {
		s, err := a.Store.Get(a.owner(w, r), r.PathValue("id"))
		if err != nil {
			a.storeError(w, err)
			return
		}
		if !s.Done() {
			fail(w, 409, "先完成当前这一轮，再查看完整结果")
			return
		}
		jsonResponse(w, 200, s.Result())
	})
	mux.HandleFunc("POST /api/sessions/{id}/follow-up", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SkillID string `json:"skillId"`
		}
		if !decode(w, r, &input) {
			return
		}
		s, err := a.Store.Update(a.owner(w, r), r.PathValue("id"), func(s *Session) error {
			if !s.Done() && len(s.FollowUpSkills) > 0 && s.FollowUpSkills[len(s.FollowUpSkills)-1] == input.SkillID {
				return nil
			}
			return s.FollowUp(input.SkillID)
		})
		if err != nil {
			a.storeError(w, err)
			return
		}
		jsonResponse(w, 200, s.View())
	})
	mux.HandleFunc("GET /api/sessions/{id}/route/{skill}", func(w http.ResponseWriter, r *http.Request) {
		s, err := a.Store.Get(a.owner(w, r), r.PathValue("id"))
		if err != nil {
			a.storeError(w, err)
			return
		}
		for _, route := range s.Content.Graph.Routes {
			if route.SkillID == r.PathValue("skill") {
				jsonResponse(w, 200, map[string]any{"route": route, "nodes": s.Content.Graph.Nodes, "version": s.Content.Graph.TaxonomyVersion})
				return
			}
		}
		fail(w, 404, "未找到知识路线")
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "未找到接口") })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := filepath.Join(a.WebDir, filepath.Clean("/"+r.URL.Path))
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			path = filepath.Join(a.WebDir, "index.html")
		}
		if _, err = os.Stat(path); err != nil {
			http.Error(w, "请先构建前端：cd frontend && npm run build", 503)
			return
		}
		http.ServeFile(w, r, path)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host {
					fail(w, 403, "请从应用页面提交")
					return
				}
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func (a *App) storeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		fail(w, 404, err.Error())
		return
	}
	// Storage errors stay on the server; content/flow errors are safe user messages.
	if strings.HasPrefix(err.Error(), "sql:") || strings.Contains(err.Error(), "database") || strings.Contains(err.Error(), "SQLite") {
		log.Printf("storage: %v", err)
		fail(w, 500, "保存暂时失败，请稍后重试")
		return
	}
	fail(w, 400, err.Error())
}
