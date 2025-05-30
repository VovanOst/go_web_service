package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const (
	OrderByAsc  = 1
	OrderByDesc = -1
	OrderByAsIs = 0
)

/*type User struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
}*/

type Users struct {
	List []User `xml:"row"`
}

type Send struct {
	PaymentApiURL string
}

type TestCase struct {
	ID            string
	SearchRequest SearchRequest
	Result        *SearchResponse
	IsError       bool
	StatusCode    int
}

type CheckoutResult struct {
	Status  int
	Balance int
	Err     string
}

func CheckoutDummy(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("id")
	switch key {
	case "42":
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status": 200, "balance": 100500}`)
	case "100500":
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status": 400, "err": "bad_balance"}`)
	case "__broken_json":
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status": 400`) //broken json
	case "__internal_error":
		fallthrough
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type Cart struct {
	PaymentApiURL string
}

func (c *Send) Checkout(id string) (*CheckoutResult, error) {
	url := c.PaymentApiURL + "?id=" + id
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := &CheckoutResult{}

	err = json.Unmarshal(data, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func runServer(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Addr:", addr, "URL:", r.URL.String())
		})

	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	fmt.Println("starting server at", addr)
	server.ListenAndServe()
}

/*
	func mainbackup() {
		cases := []TestCase{
			TestCase{
				ID: "42",
				Result: &CheckoutResult{
					Status:  200,
					Balance: 100500,
					Err:     "",
				},
				IsError: false,
			},
			TestCase{
				ID: "100500",
				Result: &CheckoutResult{
					Status:  400,
					Balance: 0,
					Err:     "bad_balance",
				},
				IsError: false,
			},
			TestCase{
				ID:      "__broken_json",
				Result:  nil,
				IsError: true,
			},
			TestCase{
				ID:      "__internal_error",
				Result:  nil,
				IsError: true,
			},
		}

		ts := httptest.NewServer(http.HandlerFunc(CheckoutDummy))
		fmt.Println("Url: %s", ts.URL)
		for caseNum, item := range cases {
			c := &Send{
				PaymentApiURL: ts.URL,
			}

			result, err := c.Checkout(item.ID)
			fmt.Println("result: %s", result)
			if err != nil && !item.IsError {
				fmt.Println("[%d] unexpected error: %#v", caseNum, err)
			}
			if err == nil && item.IsError {
				fmt.Println("[%d] expected error, got nil", caseNum)
			}
			if !reflect.DeepEqual(item.Result, result) {
				fmt.Println("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
			}
		}
		ts.Close()
		//runServer("localhost:8080")
	}
*/
func SearchServer1(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	orderBy, _ := strconv.Atoi(r.URL.Query().Get("order_by"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	//fmt.Println("%s&[%s]&[%d]&[%d]___c", query, orderField, orderBy, limit, offset)
	if orderField == "" {
		orderField = "Name"
	}

	if orderField != "id" && orderField != "Age" && orderField != "Name" {
		http.Error(w, fmt.Sprintf("unknown order_field: %s", orderField), http.StatusBadRequest)
	}

	data, err := ioutil.ReadFile("dataset.xml")
	if err != nil {
		http.Error(w, "cannot read dataset", http.StatusInternalServerError)
		return
	}

	var users Users
	if err := xml.Unmarshal(data, &users); err != nil {
		http.Error(w, "cannot parse dataset", http.StatusInternalServerError)
		return
	}
	var result []User
	for _, user := range users.List {

		name := user.FirstName + " " + user.LastName

		if query == "" || strings.Contains(name, query) || strings.Contains(user.About, query) {
			result = append(result, User{
				Id:        user.Id,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Age:       user.Age,
				About:     user.About,
			})
			fmt.Printf("Имя %s запрос %s ->прошло %v\n", name, query, strings.Contains(name, query))
		}
	}

	sort.Slice(result, func(i, j int) bool {
		switch orderField {
		case "Id":
			if orderBy == OrderByDesc {
				return result[i].Id > result[j].Id
			}
			return result[i].Id < result[j].Id
		case "Age":
			if orderBy == OrderByDesc {
				return result[i].Age > result[j].Age
			}
			return result[i].Age < result[j].Age
		case "Name":
			nameI := result[i].FirstName + " " + result[i].LastName
			nameJ := result[j].FirstName + " " + result[j].LastName
			if orderBy == OrderByDesc {
				return nameI > nameJ
			}
			return nameI < nameJ
		}
		return false
	})

	/*if offset > len(result) {
		offset = len(result)
	}*/
	if offset > len(result) {
		offset = 0
	}
	if limit+offset > len(result) {
		limit = len(result) - offset
	}

	response := result[offset : offset+limit]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/search", SearchServer1)
	http.ListenAndServe(":8080", nil)
}
