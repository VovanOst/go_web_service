// находясь в папке выше
// go build -o ./codegen.exe gen/* && ./codegen.exe pack/unkack.go  pack/marshaller.go
// go run pack/*
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"text/template"
)

type ApiMethod struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

type ApiInfo struct {
	StructName string
	Method     string
	Params     string
	Result     string
	ApiMeta    ApiMethod
}

// Шаблон для обработчиков.
// Если метод равен "GET,POST", то разрешаются оба метода, а параметры извлекаются из query.
// Если метод равен "POST", то параметры извлекаются из тела запроса.
var handlerTpl = template.Must(template.New("handlerTpl").Funcs(template.FuncMap{
	"eq": func(a, b string) bool { return a == b },
	"or": func(a, b bool) bool { return a || b },
}).Parse(`
func (srv *{{.StructName}}) handler{{.Method}}(w http.ResponseWriter, r *http.Request) {
    {{/* Проверка авторизации, если требуется */}}
    {{if .ApiMeta.Auth}}
    if r.Header.Get("X-Auth") == "" {
        w.WriteHeader(http.StatusForbidden)
        json.NewEncoder(w).Encode(map[string]interface{}{"error": "unauthorized"})
        return
    }
    {{end}}

    var in {{.Params}}

    {{if eq .ApiMeta.Method "GET,POST"}}
        if r.Method != "GET" && r.Method != "POST" {
            w.WriteHeader(http.StatusMethodNotAllowed)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad method"})
            return
        }
        if r.Method == "GET" {
            query := r.URL.Query()
            {{if eq .Params "ProfileParams"}}
                in.Login = query.Get("login")
            {{else if eq .Params "OtherCreateParams"}}
                in.Username = query.Get("username")
            {{else if eq .Params "CreateParams"}}
                in.Login = query.Get("login")
                in.Name = query.Get("full_name")
                in.Status = query.Get("status")
                in.Class = query.Get("status")
                {
                    ageStr := query.Get("age")
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
            {{end}}
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
                    {{if eq .Params "ProfileParams"}}
                        in.Login = r.FormValue("login")
                    {{else if eq .Params "OtherCreateParams"}}
                        in.Username = r.FormValue("username")
                    {{else if eq .Params "CreateParams"}}
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
                    {{end}}
                }
            } else {
                if err := r.ParseForm(); err != nil {
                    w.WriteHeader(http.StatusBadRequest)
                    json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                    return
                }
                {{if eq .Params "ProfileParams"}}
                    in.Login = r.FormValue("login")
                {{else if eq .Params "OtherCreateParams"}}
                    in.Username = r.FormValue("username")
                {{else if eq .Params "CreateParams"}}
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
                {{end}}
            }
        }
    {{else if eq .ApiMeta.Method "POST"}}
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
                {{if eq .Params "ProfileParams"}}
                    in.Login = r.FormValue("login")
                {{else if eq .Params "OtherCreateParams"}}
                    in.Username = r.FormValue("username")
                {{else if eq .Params "CreateParams"}}
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
                {{end}}
            }
        } else {
            if err := r.ParseForm(); err != nil {
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
                return
            }
            {{if eq .Params "ProfileParams"}}
                in.Login = r.FormValue("login")
            {{else if eq .Params "OtherCreateParams"}}
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
            {{else if eq .Params "CreateParams"}}
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
            {{end}}
        }
    {{else}}
        if r.Method != "{{.ApiMeta.Method}}" {
            w.WriteHeader(http.StatusMethodNotAllowed)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad method"})
            return
        }
        if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "bad request"})
            return
        }
    {{end}}

    {{if or (eq .Params "ProfileParams") (eq .Params "CreateParams")}}
        if in.Login == "" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "login must me not empty"})
            return
        }
       
    {{end}}

    {{if eq .Params "CreateParams"}}
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
    {{end}}

      {{if eq .Params "OtherCreateParams"}}
        if in.Class != "warrior" && in.Class != "sorcerer" && in.Class != "rouge" {
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]interface{}{"error": "class must be one of [warrior, sorcerer, rouge]"})
            return
        }
    {{end}}

    res, err := srv.{{.Method}}(r.Context(), in)
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
`))

// Шаблон для ServeHTTP с нормализацией URL.
var serveHTTPTpl = template.Must(template.New("serveHTTPTpl").Parse(`
func (srv *{{.StructName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch strings.TrimRight(r.URL.Path, "/") {
    {{range .Methods}}
    case "{{.ApiMeta.Url}}":
        srv.handler{{.Method}}(w, r)
    {{end}}
    default:
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(map[string]interface{}{
                "error": "unknown method",
            })
            return
    }
}
`))

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(out, "package "+node.Name.Name)
	fmt.Fprintln(out) // пустая строка
	fmt.Fprintln(out, "import (")
	fmt.Fprintln(out, `"bytes"`)
	fmt.Fprintln(out, `"encoding/json"`)
	fmt.Fprintln(out, `"strconv"`)
	fmt.Fprintln(out, `"net/http"`)
	fmt.Fprintln(out, `"io/ioutil"`)
	fmt.Fprintln(out, `"strings"`)
	fmt.Fprintln(out, ")")
	fmt.Fprintln(out) // пустая строка

	apiMethods := make(map[string][]ApiInfo)

	for _, decl := range node.Decls {
		// Ищем методы структур (FuncDecl)
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || funcDecl.Doc == nil {
			continue
		}

		// Ищем комментарий с аннотацией apigen:api
		var apiMeta ApiMethod
		for _, comment := range funcDecl.Doc.List {
			if strings.HasPrefix(comment.Text, "// apigen:api") {
				jsonStr := strings.TrimPrefix(comment.Text, "// apigen:api ")
				if err := json.Unmarshal([]byte(jsonStr), &apiMeta); err != nil {
					log.Fatalf("Ошибка разбора JSON в %s: %v", funcDecl.Name.Name, err)
				}
			}
		}

		if apiMeta.Url == "" {
			continue
		}

		// Если HTTP-метод не задан, подставляем "GET,POST"
		if apiMeta.Method == "" {
			apiMeta.Method = "GET,POST"
		}

		// Определяем, к какой структуре относится метод
		structName := funcDecl.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name

		// Проверяем, что метод имеет хотя бы 2 аргумента (context и params)
		if len(funcDecl.Type.Params.List) < 2 {
			log.Fatalf("Метод %s должен иметь 2 аргумента (context, params)", funcDecl.Name.Name)
		}
		paramsType := funcDecl.Type.Params.List[1].Type.(*ast.Ident).Name
		resultType := funcDecl.Type.Results.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name

		apiMethods[structName] = append(apiMethods[structName], ApiInfo{
			StructName: structName,
			Method:     funcDecl.Name.Name,
			Params:     paramsType,
			Result:     resultType,
			ApiMeta:    apiMeta,
		})
	}
	fmt.Printf("type: %T data: %+v\n", apiMethods, apiMethods)

	// Генерируем код обработчиков и ServeHTTP для каждой структуры
	for structName, methods := range apiMethods {
		for _, method := range methods {
			handlerTpl.Execute(out, method)
		}
		serveHTTPTpl.Execute(out, struct {
			StructName string
			Methods    []ApiInfo
		}{structName, methods})
	}

	fmt.Println("Code generation complete")
}

// go build gen/* && ./codegen.exe pack/unpack.go  pack/marshaller.go
// go run pack/*
