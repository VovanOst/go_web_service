package main

import (
"bytes"
"encoding/json"
"strconv"
"net/http"
"io/ioutil"
"strings"
)


func (srv *OtherApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
    
    
    if r.Header.Get("X-Auth") == "" {
        w.WriteHeader(http.StatusForbidden)
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "unauthorized"})
        return
    }
    

    var in OtherCreateParams

    
        if r.Method != "POST" {
            w.WriteHeader(http.StatusNotAcceptable)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad method"})
            return
        }
        if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") && r.ContentLength > 0 {
            bodyBytes, err := ioutil.ReadAll(r.Body)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                return
            }
            r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
            trimmedBody := bytes.TrimSpace(bodyBytes)
            if len(trimmedBody) > 0 && trimmedBody[0] == '{' {
                if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                    return
                }
            } else {
                if err := r.ParseForm(); err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                    return
                }
                
                    in.Username = r.FormValue("username")
                
            }
        } else {
            if err := r.ParseForm(); err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                return
            }
            
                in.Username = r.FormValue("username")
                in.Class = r.FormValue("class")
                {
                    levelStr := r.FormValue("level")
                    if levelStr != "" {
                        l, err := strconv.Atoi(levelStr)
                        if err != nil {
                            w.WriteHeader(http.StatusBadRequest)
                            json.NewEncoder(w).Encode(map[string]interface{}{"error": "level must be int"})
                            return
                        }
                        in.Level = l
                    }
                }
                in.Name = r.FormValue("account_name")
            
        }
    

    

    

      
        if in.Class != "warrior" && in.Class != "sorcerer" && in.Class != "rouge" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "class must be one of [warrior, sorcerer, rouge]"})
            return
        }
    

    res, err := srv.Create(r.Context(), in)
    if err != nil {
        if apiErr, ok := err.(ApiError); ok {
            w.WriteHeader(apiErr.HTTPStatus)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": apiErr.Error()})
        } else {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
        }
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{"error": "", "response": res})
}

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch strings.TrimRight(r.URL.Path, "/") {
    
    case "/user/create":
        srv.handlerCreate(w, r)
    
    default:
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "error": "unknown method",
            })
            return
    }
}

func (srv *MyApi) handlerProfile(w http.ResponseWriter, r *http.Request) {
    
    

    var in ProfileParams

    
        if r.Method != "GET" && r.Method != "POST" {
            w.WriteHeader(http.StatusMethodNotAllowed)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad method"})
            return
        }
        if r.Method == "GET" {
            query := r.URL.Query()
            
                in.Login = query.Get("login")
            
        } else { // POST
            contentType := r.Header.Get("Content-Type")
            if strings.HasPrefix(contentType, "application/json") && r.ContentLength > 0 {
                bodyBytes, err := ioutil.ReadAll(r.Body)
                if err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                    return
                }
                r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
                trimmedBody := bytes.TrimSpace(bodyBytes)
                if len(trimmedBody) > 0 && trimmedBody[0] == '{' {
                    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
                        w.WriteHeader(http.StatusBadRequest)
                        json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                        return
                    }
                } else {
                    if err := r.ParseForm(); err != nil {
                        w.WriteHeader(http.StatusBadRequest)
                        json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                        return
                    }
                    
                        in.Login = r.FormValue("login")
                    
                }
            } else {
                if err := r.ParseForm(); err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                    return
                }
                
                    in.Login = r.FormValue("login")
                
            }
        }
    

    
        if in.Login == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "login must me not empty"})
            return
        }
       
    

    

      

    res, err := srv.Profile(r.Context(), in)
    if err != nil {
        if apiErr, ok := err.(ApiError); ok {
            w.WriteHeader(apiErr.HTTPStatus)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": apiErr.Error()})
        } else {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
        }
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{"error": "", "response": res})
}

func (srv *MyApi) handlerCreate(w http.ResponseWriter, r *http.Request) {
    
    
    if r.Header.Get("X-Auth") == "" {
        w.WriteHeader(http.StatusForbidden)
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "unauthorized"})
        return
    }
    

    var in CreateParams

    
        if r.Method != "POST" {
            w.WriteHeader(http.StatusNotAcceptable)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad method"})
            return
        }
        if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") && r.ContentLength > 0 {
            bodyBytes, err := ioutil.ReadAll(r.Body)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                return
            }
            r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
            trimmedBody := bytes.TrimSpace(bodyBytes)
            if len(trimmedBody) > 0 && trimmedBody[0] == '{' {
                if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                    return
                }
            } else {
                if err := r.ParseForm(); err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                    return
                }
                
                    in.Login = r.FormValue("login")
                    in.Name = r.FormValue("full_name")
                    in.Status = r.FormValue("status")
                    {
                        ageStr := r.FormValue("age")
                        if ageStr != "" {
                            a, err := strconv.Atoi(ageStr)
                            if err != nil {
                                w.WriteHeader(http.StatusBadRequest)
                                json.NewEncoder(w).Encode(map[string]interface{}{"error": "age must be int"})
                                return
                            }
                            in.Age = a
                        }
                    }
                
            }
        } else {
            if err := r.ParseForm(); err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                return
            }
            
                in.Login = r.FormValue("login")
                in.Name = r.FormValue("full_name")
                in.Status = r.FormValue("status")
                {
                    ageStr := r.FormValue("age")
                    if ageStr != "" {
                        a, err := strconv.Atoi(ageStr)
                        if err != nil {
                            w.WriteHeader(http.StatusBadRequest)
                            json.NewEncoder(w).Encode(map[string]interface{}{"error": "age must be int"})
                            return
                        }
                        in.Age = a
                    }
                }
                if in.Status == "" {
                    in.Status = "user"
                }
            
        }
    

    
        if in.Login == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "login must me not empty"})
            return
        }
       
    

    
        if len(in.Login) < 10 {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "login len must be >= 10"})
            return
        }
        if in.Age < 0 {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "age must be >= 0"})
            return
        }
        if in.Age > 128 {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "age must be <= 128"})
            return
        }
        
        if in.Status != "user" && in.Status != "moderator" && in.Status != "admin" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "status must be one of [user, moderator, admin]"})
            return
        }
        
        if in.Status != "user" && in.Status != "moderator" && in.Status != "admin" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "status must be one of [user, moderator, admin]"})
            return
        }
    

      

    res, err := srv.Create(r.Context(), in)
    if err != nil {
        if apiErr, ok := err.(ApiError); ok {
            w.WriteHeader(apiErr.HTTPStatus)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": apiErr.Error()})
        } else {
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
        }
        return
    }
    
    json.NewEncoder(w).Encode(map[string]interface{}{"error": "", "response": res})
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch strings.TrimRight(r.URL.Path, "/") {
    
    case "/user/profile":
        srv.handlerProfile(w, r)
    
    case "/user/create":
        srv.handlerCreate(w, r)
    
    default:
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "error": "unknown method",
            })
            return
    }
}
