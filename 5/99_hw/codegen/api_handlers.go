package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (srv *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
	var in ProfileParams

	// Для GET-запросов извлекаем параметры из URL
	if r.Method == "GET" {
		query := r.URL.Query()
		in.Login = query.Get("login")
		// Если нужны дополнительные поля, извлекайте и конвертируйте их
	} else if r.Method == "POST" {
		// Для POST ожидаем JSON
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}

	res, err := srv.Profile(r.Context(), in)
	if err != nil {
		if apiErr, ok := err.(ApiError); ok {
			http.Error(w, apiErr.Error(), apiErr.HTTPStatus)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (srv *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}

	var in CreateParams
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		// При JSON-запросе используем декодер
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	} else {
		// Предполагаем, что данные приходят в виде URL-encoded формы
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		in.Login = r.FormValue("login")
		in.Name = r.FormValue("full_name")
		in.Status = r.FormValue("status")
		ageStr := r.FormValue("age")
		if ageStr != "" {
			age, err := strconv.Atoi(ageStr)
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			in.Age = age
		}
	}

	res, err := srv.Create(r.Context(), in)
	if err != nil {
		if apiErr, ok := err.(ApiError); ok {
			http.Error(w, apiErr.Error(), apiErr.HTTPStatus)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/profile":
		srv.handlerProfile(w, r)
	case "/user/create":
		srv.handlerCreate(w, r)
	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}
}

func (srv *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}

	var in OtherCreateParams
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
	} else {
		// Обработка URL-encoded формы
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		in.Username = r.FormValue("username")
		in.Name = r.FormValue("account_name")
		in.Class = r.FormValue("class")
		levelStr := r.FormValue("level")
		if levelStr != "" {
			level, err := strconv.Atoi(levelStr)
			if err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			in.Level = level
		}
	}

	res, err := srv.Create(r.Context(), in)
	if err != nil {
		if apiErr, ok := err.(ApiError); ok {
			http.Error(w, apiErr.Error(), apiErr.HTTPStatus)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		srv.handlerCreate(w, r)
	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}
}
