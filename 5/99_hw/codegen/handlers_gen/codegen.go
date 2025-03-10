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
type tpl struct {
	FieldName string
}

// Шаблон для обработчиков
var handlerTpl = template.Must(template.New("handlerTpl").Parse(`
func (srv *{{.StructName}}) handler{{.Method}}(w http.ResponseWriter, r *http.Request) {
	if r.Method != "{{.ApiMeta.Method}}" {
		http.Error(w, "bad method", http.StatusMethodNotAllowed)
		return
	}

	var in {{.Params}}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	res, err := srv.{{.Method}}(r.Context(), in)
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
`))

// Шаблон для ServeHTTP
var serveHTTPTpl = template.Must(template.New("serveHTTPTpl").Parse(`
func (srv *{{.StructName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
		{{range .Methods}}
		case "{{.ApiMeta.Url}}":
			srv.handler{{.Method}}(w, r)
		{{end}}
	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}
}
`))

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])

	fmt.Fprintln(out, `package `+node.Name.Name)
	fmt.Fprintln(out) // empty line
	fmt.Fprintln(out, "import (")
	fmt.Fprintln(out, `"net/http"`)
	fmt.Fprintln(out, `"encoding/json"`)
	fmt.Fprintln(out, ")\n")

	fmt.Fprintln(out) // empty line

	apiMethods := make(map[string][]ApiInfo)

	for _, decl := range node.Decls {
		// Ищем методы структур (FuncDecl)
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil || funcDecl.Doc == nil {
			continue
		}

		// Проверяем, есть ли у метода комментарий `apigen:api`
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

		// Определяем структуру, к которой относится метод
		structName := funcDecl.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name

		// Получаем параметры метода
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

	// Генерируем код
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
